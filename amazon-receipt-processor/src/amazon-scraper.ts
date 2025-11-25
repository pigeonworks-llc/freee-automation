/**
 * Amazon Receipt Scraper using Playwright
 * Downloads receipt PDFs from Amazon order history
 */

import { chromium, Browser, BrowserContext, Page } from 'playwright';
import * as path from 'path';
import * as fs from 'fs';
import dayjs from 'dayjs';

export interface ScraperConfig {
  userDataDir: string;
  downloadDir: string;
  headless: boolean;
}

export interface DownloadResult {
  orderId: string;
  orderDate: string;
  amount: number;
  filePath: string;
  status: 'downloaded' | 'skipped' | 'error';
  error?: string;
}

export interface OrderInfo {
  orderId: string;
  orderDate: string;
  amount: number;
  receiptUrl: string;
  orderDetailUrl: string;
  paymentMethod?: string;
}

const AMAZON_ORDER_HISTORY_URL = 'https://www.amazon.co.jp/gp/your-account/order-history';
const DOWNLOAD_DELAY_MS = 2000;
const DEBUG_HTML = false; // Set to true to capture HTML for debugging

export class AmazonScraper {
  private config: ScraperConfig;
  private browser: Browser | null = null;
  private context: BrowserContext | null = null;

  constructor(config: ScraperConfig) {
    this.config = config;
    // Ensure directories exist
    if (!fs.existsSync(config.userDataDir)) {
      fs.mkdirSync(config.userDataDir, { recursive: true });
    }
    if (!fs.existsSync(config.downloadDir)) {
      fs.mkdirSync(config.downloadDir, { recursive: true });
    }
  }

  async initialize(): Promise<void> {
    console.log('Starting browser...');

    this.browser = await chromium.launch({
      headless: this.config.headless,
    });

    this.context = await this.browser.newContext({
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      locale: 'ja-JP',
      viewport: { width: 1280, height: 800 },
    });

    // Load cookies if saved
    const cookiePath = path.join(this.config.userDataDir, 'cookies.json');
    if (fs.existsSync(cookiePath)) {
      const cookies = JSON.parse(fs.readFileSync(cookiePath, 'utf-8'));
      await this.context.addCookies(cookies);
      console.log('Loaded saved session');
    }
  }

  async close(): Promise<void> {
    if (this.context) {
      // Save cookies for next session
      const cookies = await this.context.cookies();
      const cookiePath = path.join(this.config.userDataDir, 'cookies.json');
      fs.writeFileSync(cookiePath, JSON.stringify(cookies, null, 2));
      console.log('Session saved');
    }
    if (this.browser) {
      await this.browser.close();
    }
  }

  async ensureLoggedIn(page: Page): Promise<boolean> {
    // Navigate to order history
    await page.goto(AMAZON_ORDER_HISTORY_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });

    // Check if we need to login
    const loginForm = await page.$('#ap_email');
    if (loginForm) {
      console.log('');
      console.log('Login required. Please log in manually in the browser window.');
      console.log('The browser will wait for you to complete the login...');
      console.log('');

      // Wait for navigation to order history (indicates successful login)
      try {
        await page.waitForURL(/order-history/, { timeout: 300000 }); // 5 minutes timeout
        console.log('Login successful!');

        // Save cookies after login
        const cookies = await this.context!.cookies();
        const cookiePath = path.join(this.config.userDataDir, 'cookies.json');
        fs.writeFileSync(cookiePath, JSON.stringify(cookies, null, 2));

        return true;
      } catch {
        console.error('Login timeout or failed');
        return false;
      }
    }

    // Already logged in
    console.log('Already logged in');
    return true;
  }

  async getOrdersInRange(page: Page, fromDate: string, toDate: string): Promise<OrderInfo[]> {
    const orders: OrderInfo[] = [];
    const from = dayjs(fromDate);
    const to = dayjs(toDate);

    // Get unique years from the date range
    const years = new Set<number>();
    let current = from;
    while (current.isBefore(to) || current.isSame(to, 'day')) {
      years.add(current.year());
      current = current.add(1, 'month');
    }

    for (const year of Array.from(years).sort()) {
      console.log(`Scanning orders from ${year}...`);

      // Navigate to specific year
      const yearUrl = `${AMAZON_ORDER_HISTORY_URL}?orderFilter=year-${year}`;
      await page.goto(yearUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
      await this.delay(2000);

      let hasNextPage = true;
      let pageNum = 1;

      while (hasNextPage) {
        console.log(`  Page ${pageNum}...`);

        // Debug: Save HTML for analysis
        if (DEBUG_HTML && pageNum === 1) {
          const html = await page.content();
          const debugPath = path.join(this.config.userDataDir, `debug-orders-${year}.html`);
          fs.writeFileSync(debugPath, html);
          console.log(`    [DEBUG] Saved HTML to ${debugPath}`);
        }

        // Find all order cards - try multiple selectors
        const orderCards = await page.$$('.order-card, .order, [data-component="order"], .a-box-group.order');

        if (DEBUG_HTML) {
          console.log(`    [DEBUG] Found ${orderCards.length} order cards with primary selector`);
          if (orderCards.length === 0) {
            // Try alternative selectors for debugging
            const altSelectors = [
              '.js-order-card',
              '.a-box-group',
              '[data-oor-id]',
              '.order-info',
            ];
            for (const sel of altSelectors) {
              const count = await page.$$(sel).then(els => els.length);
              if (count > 0) {
                console.log(`    [DEBUG] Found ${count} elements with selector: ${sel}`);
              }
            }
          }
        }

        for (const card of orderCards) {
          try {
            const orderInfo = await this.parseOrderCard(page, card);
            if (!orderInfo) {
              if (DEBUG_HTML) {
                const cardHtml = await card.innerHTML();
                console.log(`    [DEBUG] Failed to parse card, HTML snippet: ${cardHtml.substring(0, 200)}...`);
              }
              continue;
            }

            const orderDate = dayjs(orderInfo.orderDate);

            // Check if within date range
            if (orderDate.isBefore(from, 'day') || orderDate.isAfter(to, 'day')) {
              continue;
            }

            orders.push(orderInfo);
          } catch (e) {
            // Skip orders that can't be parsed
          }
        }

        // Check for next page
        const nextButton = await page.$('li.a-last:not(.a-disabled) a');
        if (nextButton) {
          await nextButton.click();
          await page.waitForLoadState('domcontentloaded');
          await this.delay(2000);
          pageNum++;
        } else {
          hasNextPage = false;
        }
      }
    }

    console.log(`Found ${orders.length} orders in date range`);

    // Also scan digital orders
    console.log('Scanning digital orders...');
    const digitalOrders = await this.getDigitalOrders(page, from, to);
    orders.push(...digitalOrders);
    console.log(`Total orders (including digital): ${orders.length}`);

    return orders;
  }

  private async getDigitalOrders(page: Page, from: dayjs.Dayjs, to: dayjs.Dayjs): Promise<OrderInfo[]> {
    const orders: OrderInfo[] = [];

    // Navigate to digital orders page
    const digitalUrl = `${AMAZON_ORDER_HISTORY_URL}?digitalOrders=1&unifiedOrders=0`;
    await page.goto(digitalUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await this.delay(2000);

    let hasNextPage = true;
    let pageNum = 1;

    while (hasNextPage && pageNum <= 10) { // Limit to 10 pages for digital orders
      console.log(`  Digital page ${pageNum}...`);

      // Debug: Save HTML for analysis
      if (DEBUG_HTML && pageNum === 1) {
        const html = await page.content();
        const debugPath = path.join(this.config.userDataDir, 'debug-digital-orders.html');
        fs.writeFileSync(debugPath, html);
        console.log(`    [DEBUG] Saved digital orders HTML to ${debugPath}`);
      }

      // Digital orders have different structure - look for order rows
      const orderRows = await page.$$('.order-card, .a-box-group, [id^="orderCard"]');

      if (DEBUG_HTML) {
        console.log(`    [DEBUG] Found ${orderRows.length} digital order rows`);
      }

      for (const row of orderRows) {
        try {
          const orderInfo = await this.parseDigitalOrderCard(page, row);
          if (!orderInfo) continue;

          const orderDate = dayjs(orderInfo.orderDate);
          if (orderDate.isBefore(from, 'day')) {
            // Digital orders are sorted by date descending, so we can stop early
            hasNextPage = false;
            break;
          }
          if (orderDate.isAfter(to, 'day')) continue;

          orders.push(orderInfo);
        } catch {
          // Skip orders that can't be parsed
        }
      }

      if (!hasNextPage) break;

      // Check for next page
      const nextButton = await page.$('li.a-last:not(.a-disabled) a');
      if (nextButton) {
        await nextButton.click();
        await page.waitForLoadState('domcontentloaded');
        await this.delay(2000);
        pageNum++;
      } else {
        hasNextPage = false;
      }
    }

    console.log(`  Found ${orders.length} digital orders in date range`);
    return orders;
  }

  private async parseDigitalOrderCard(page: Page, card: any): Promise<OrderInfo | null> {
    try {
      const cardText = await card.textContent() || '';

      // Extract order ID
      const orderIdMatch = cardText.match(/(\d{3}-\d{7}-\d{7})|D\d{2}-\d{7}-\d{7}/);
      if (!orderIdMatch) return null;
      const orderId = orderIdMatch[0];

      // Extract date
      const orderDate = this.parseJapaneseDate(cardText);
      if (!orderDate) return null;

      // Extract amount - digital orders may show amount differently
      let amount = 0;
      const amountMatch = cardText.match(/[¥￥]\s*([\d,]+)/);
      if (amountMatch) {
        amount = this.parseAmount(amountMatch[1]);
      }

      if (DEBUG_HTML && amount === 0) {
        console.log(`    [DEBUG] Digital order amount extraction failed for ${orderId}`);
        console.log(`    [DEBUG] Card text: ${cardText.substring(0, 300)}...`);
      }

      // Build URLs
      const orderDetailUrl = `https://www.amazon.co.jp/gp/your-account/order-details?orderID=${orderId}`;
      const receiptUrl = `https://www.amazon.co.jp/gp/digital/your-account/order-summary.html?orderID=${orderId}`;

      if (DEBUG_HTML) {
        console.log(`    [DEBUG] Digital order: ${orderId}, ${orderDate}, ¥${amount}`);
      }

      return {
        orderId,
        orderDate,
        amount,
        receiptUrl,
        orderDetailUrl,
      };
    } catch {
      return null;
    }
  }

  private async parseOrderCard(page: Page, card: any): Promise<OrderInfo | null> {
    try {
      // Get all text content for debugging and searching
      const cardText = await card.textContent() || '';

      // Extract order ID - try multiple patterns
      let orderId: string | null = null;

      // Try specific selectors first
      const orderIdSelectors = [
        '.yohtmlc-order-id span:last-child',
        '[data-a-color="secondary"]',
        '.a-color-secondary',
        'span[dir="ltr"]',
      ];

      for (const sel of orderIdSelectors) {
        const el = await card.$(sel);
        if (el) {
          const text = await el.textContent();
          const match = text?.match(/(\d{3}-\d{7}-\d{7})/);
          if (match) {
            orderId = match[1];
            break;
          }
        }
      }

      // Fallback: search in all text
      if (!orderId) {
        const match = cardText.match(/(\d{3}-\d{7}-\d{7})/);
        if (match) orderId = match[1];
      }

      if (!orderId) {
        if (DEBUG_HTML) {
          console.log(`    [DEBUG] No order ID found in card text: ${cardText.substring(0, 100)}...`);
        }
        return null;
      }

      // Extract date - search in text
      const orderDate = this.parseJapaneseDate(cardText);
      if (!orderDate) {
        if (DEBUG_HTML) {
          console.log(`    [DEBUG] No date found for order ${orderId}`);
        }
        return null;
      }

      // Extract amount - search for price pattern in text
      const amountMatch = cardText.match(/[¥￥][\s]*([\d,]+)/);
      const amount = amountMatch ? this.parseAmount(amountMatch[1]) : 0;
      if (DEBUG_HTML && amount === 0) {
        console.log(`    [DEBUG] Amount extraction failed for ${orderId}`);
        console.log(`    [DEBUG] Card text: ${cardText.substring(0, 500)}...`);
      }

      // Build order detail URL from order ID
      const orderDetailUrl = `https://www.amazon.co.jp/gp/your-account/order-details?orderID=${orderId}`;

      // Find receipt/invoice link - try multiple selectors
      const receiptSelectors = [
        'a[href*="invoice"]',
        'a[href*="receipt"]',
        'a:has-text("領収書")',
      ];

      let receiptUrl = '';
      for (const sel of receiptSelectors) {
        try {
          const link = await card.$(sel);
          if (link) {
            receiptUrl = await link.getAttribute('href') || '';
            if (receiptUrl) {
              if (!receiptUrl.startsWith('http')) {
                receiptUrl = 'https://www.amazon.co.jp' + receiptUrl;
              }
              break;
            }
          }
        } catch {
          // Selector might not be valid, continue
        }
      }

      // If no receipt URL found, use order details as fallback
      if (!receiptUrl) {
        receiptUrl = orderDetailUrl;
      }

      if (DEBUG_HTML) {
        console.log(`    [DEBUG] Parsed: ${orderId}, ${orderDate}, ¥${amount}`);
      }

      return {
        orderId,
        orderDate,
        amount,
        receiptUrl,
        orderDetailUrl,
      };
    } catch (e) {
      if (DEBUG_HTML) {
        console.log(`    [DEBUG] Parse error: ${e}`);
      }
      return null;
    }
  }

  private parseJapaneseDate(text: string): string | null {
    // Match Japanese date format: 2024年1月15日
    const match = text.match(/(\d{4})年(\d{1,2})月(\d{1,2})日/);
    if (match) {
      const year = match[1];
      const month = match[2].padStart(2, '0');
      const day = match[3].padStart(2, '0');
      return `${year}-${month}-${day}`;
    }
    return null;
  }

  private parseAmount(text: string): number {
    const cleaned = text.replace(/[^\d]/g, '');
    return parseInt(cleaned, 10) || 0;
  }

  /**
   * Navigate to order detail page and extract payment method
   */
  async getPaymentMethodFromDetailPage(page: Page, orderDetailUrl: string): Promise<string | undefined> {
    try {
      await page.goto(orderDetailUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
      await this.delay(1500);

      // Get full page text for searching
      const pageText = await page.textContent('body') || '';

      // Debug: save HTML if enabled
      if (DEBUG_HTML) {
        const html = await page.content();
        const debugPath = path.join(this.config.userDataDir, 'debug-order-detail.html');
        fs.writeFileSync(debugPath, html);
        console.log(`      [DEBUG] Saved order detail HTML to ${debugPath}`);
      }

      // Look for payment method patterns
      const paymentPatterns = [
        // Card type with last 4 digits (English)
        /(?:Visa|Mastercard|MasterCard|JCB|AMEX|American Express|Diners|Discover)[^\d]*?(?:末尾|ending in|ending|\*{4})?[^\d]*?(\d{4})/i,
        // Japanese patterns
        /クレジットカード[：:]*\s*([^\s]+末尾\d{4})/,
        /(?:お支払い方法|支払い方法)[：:]*\s*([^\n]+)/,
        // Card ending pattern
        /末尾\s*(\d{4})[^\d]/,
        // Generic card pattern
        /\*{4}\s*(\d{4})/,
      ];

      for (const pattern of paymentPatterns) {
        const match = pageText.match(pattern);
        if (match) {
          // Clean up the match
          let paymentMethod = match[0].trim();
          // Remove excess whitespace
          paymentMethod = paymentMethod.replace(/\s+/g, ' ');
          if (DEBUG_HTML) {
            console.log(`      [DEBUG] Found payment method: ${paymentMethod}`);
          }
          return paymentMethod;
        }
      }

      // Try to find payment section specifically
      const paymentSection = await page.$('[data-component="paymentMethod"], .payment-method, #od-subtotals');
      if (paymentSection) {
        const sectionText = await paymentSection.textContent() || '';
        // Look for any card pattern in the section
        const cardMatch = sectionText.match(/(Visa|Mastercard|MasterCard|JCB|AMEX|American Express|Diners)[^\n]*/i);
        if (cardMatch) {
          return cardMatch[0].trim();
        }
      }

      if (DEBUG_HTML) {
        console.log(`      [DEBUG] No payment method found in page`);
      }
      return undefined;
    } catch (error) {
      if (DEBUG_HTML) {
        console.log(`      [DEBUG] Error fetching payment method: ${error}`);
      }
      return undefined;
    }
  }

  async downloadReceipt(page: Page, order: OrderInfo): Promise<DownloadResult> {
    const filename = `領収書-${order.orderId}.pdf`;
    const filePath = path.join(this.config.downloadDir, filename);

    // Check if already downloaded
    if (fs.existsSync(filePath)) {
      return {
        orderId: order.orderId,
        orderDate: order.orderDate,
        amount: order.amount,
        filePath,
        status: 'skipped',
      };
    }

    try {
      // Method 1: Try to download invoice from order detail page via UI
      const pdfBuffer = await this.downloadInvoicePdf(page, order.orderDetailUrl);
      if (pdfBuffer) {
        fs.writeFileSync(filePath, pdfBuffer);
        console.log(`    Downloaded: ${filename}`);
        return {
          orderId: order.orderId,
          orderDate: order.orderDate,
          amount: order.amount,
          filePath,
          status: 'downloaded',
        };
      }

      // Method 2: Fallback to receiptUrl if available
      if (order.receiptUrl) {
        await page.goto(order.receiptUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
        await this.delay(2000);

        // Save page as PDF
        await page.pdf({ path: filePath, format: 'A4' });

        return {
          orderId: order.orderId,
          orderDate: order.orderDate,
          amount: order.amount,
          filePath,
          status: 'downloaded',
        };
      }

      return {
        orderId: order.orderId,
        orderDate: order.orderDate,
        amount: order.amount,
        filePath: '',
        status: 'error',
        error: 'Could not download invoice',
      };
    } catch (error: any) {
      return {
        orderId: order.orderId,
        orderDate: order.orderDate,
        amount: order.amount,
        filePath: '',
        status: 'error',
        error: error.message || String(error),
      };
    }
  }

  /**
   * Download invoice PDF from order detail page by clicking UI elements
   * This works for both physical and digital orders (Kindle, Prime Video, etc.)
   */
  private async downloadInvoicePdf(page: Page, orderDetailUrl: string): Promise<Buffer | null> {
    try {
      await page.goto(orderDetailUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
      await this.delay(2000);

      // Look for "領収書等" dropdown
      const receiptDropdown = page.locator('text=領収書等').first();
      if (!await receiptDropdown.isVisible({ timeout: 5000 }).catch(() => false)) {
        console.log('    Receipt dropdown not found on detail page');
        return null;
      }

      await receiptDropdown.click();
      await this.delay(1000);

      // Look for "明細書／適格請求書" link
      const invoiceLink = page.locator('text=明細書／適格請求書').first();
      if (!await invoiceLink.isVisible({ timeout: 3000 }).catch(() => false)) {
        console.log('    Invoice link not found');
        return null;
      }

      // Set up response interception to capture PDF URL
      let pdfUrl: string | null = null;

      const responseHandler = async (response: any) => {
        const responseUrl = response.url();
        if (responseUrl.includes('documents/download') && responseUrl.includes('invoice')) {
          pdfUrl = responseUrl;
        }
      };

      page.on('response', responseHandler);

      // Click the invoice link
      await invoiceLink.click();
      await this.delay(3000);

      page.off('response', responseHandler);

      if (!pdfUrl) {
        // Check if page navigated to PDF URL
        const currentUrl = page.url();
        if (currentUrl.includes('documents/download') && currentUrl.includes('invoice')) {
          pdfUrl = currentUrl;
        }
      }

      if (!pdfUrl) {
        console.log('    Could not capture PDF URL');
        return null;
      }

      // Fetch PDF using page context with Accept header
      const pdfData = await page.evaluate(async (url: string) => {
        const response = await fetch(url, {
          headers: {
            'Accept': 'application/pdf'
          },
          credentials: 'include'
        });
        const arrayBuffer = await response.arrayBuffer();
        return Array.from(new Uint8Array(arrayBuffer));
      }, pdfUrl);

      const buffer = Buffer.from(pdfData);

      // Verify it's a valid PDF
      if (buffer.length > 100 && buffer.toString('utf8', 0, 4) === '%PDF') {
        return buffer;
      }

      console.log('    Fetched content is not a valid PDF');
      return null;
    } catch (error: any) {
      console.log(`    Error downloading invoice: ${error.message}`);
      return null;
    }
  }

  async downloadReceipts(fromDate: string, toDate: string, dryRun = false, cardFilter?: string): Promise<DownloadResult[]> {
    if (!this.context) {
      throw new Error('Browser not initialized. Call initialize() first.');
    }

    const page = await this.context.newPage();
    const results: DownloadResult[] = [];

    try {
      // Ensure logged in
      const loggedIn = await this.ensureLoggedIn(page);
      if (!loggedIn) {
        throw new Error('Failed to log in to Amazon');
      }

      // Get orders in date range
      let orders = await this.getOrdersInRange(page, fromDate, toDate);

      if (orders.length === 0) {
        console.log('No orders found in the specified date range');
        return results;
      }

      console.log(`Found ${orders.length} orders in date range`);

      // If card filter is specified, fetch payment info from detail pages
      if (cardFilter) {
        console.log(`Fetching payment info to filter by card "${cardFilter}"...`);
        console.log('(This may take a while as we need to visit each order detail page)');

        for (let i = 0; i < orders.length; i++) {
          const order = orders[i];
          console.log(`  [${i + 1}/${orders.length}] Checking ${order.orderId}...`);

          const paymentMethod = await this.getPaymentMethodFromDetailPage(page, order.orderDetailUrl);
          order.paymentMethod = paymentMethod;

          if (paymentMethod) {
            console.log(`    Payment: ${paymentMethod}`);
          } else {
            console.log(`    Payment: (not found)`);
          }

          // Rate limiting between detail page visits
          await this.delay(1500);
        }

        // Filter by card
        const beforeCount = orders.length;
        orders = orders.filter(o => {
          if (!o.paymentMethod) return false;
          return o.paymentMethod.toLowerCase().includes(cardFilter.toLowerCase());
        });
        console.log(`Filtered to ${orders.length} orders matching card "${cardFilter}" (from ${beforeCount})`);
      }

      if (orders.length === 0) {
        console.log('No orders match the card filter');
        return results;
      }

      console.log(`${orders.length} orders to download`);

      // Download each receipt
      for (let i = 0; i < orders.length; i++) {
        const order = orders[i];
        console.log(`[${i + 1}/${orders.length}] Order ${order.orderId} (${order.orderDate})`);

        if (dryRun) {
          console.log('    [DRY RUN] Would download receipt');
          results.push({
            orderId: order.orderId,
            orderDate: order.orderDate,
            amount: order.amount,
            filePath: '',
            status: 'skipped',
          });
        } else {
          const result = await this.downloadReceipt(page, order);
          results.push(result);

          if (result.status === 'downloaded') {
            // Rate limiting
            await this.delay(DOWNLOAD_DELAY_MS);
          }
        }
      }
    } finally {
      await page.close();
    }

    return results;
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
