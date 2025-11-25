/**
 * Amazon Receipt PDF Parser
 * Extracts order information from Amazon receipt PDFs
 */

import pdfParse from 'pdf-parse';
import * as fs from 'fs';

export interface AmazonReceiptData {
  orderNumber: string;
  orderDate: string; // YYYY-MM-DD format
  totalAmount: number;
  fileName: string;
}

/**
 * Parse Amazon receipt PDF and extract key information
 */
export async function parseAmazonReceipt(pdfPath: string): Promise<AmazonReceiptData | null> {
  try {
    const dataBuffer = fs.readFileSync(pdfPath);
    const pdfData = await pdfParse(dataBuffer);
    const text = pdfData.text;

    // Extract order number (e.g., 123-4567890-1234567)
    const orderNumber = extractOrderNumber(text);
    if (!orderNumber) {
      console.warn(`Could not extract order number from ${pdfPath}`);
      return null;
    }

    // Extract order date
    const orderDate = extractOrderDate(text);
    if (!orderDate) {
      console.warn(`Could not extract order date from ${pdfPath}`);
      return null;
    }

    // Extract total amount
    const totalAmount = extractTotalAmount(text);
    if (totalAmount === null) {
      console.warn(`Could not extract total amount from ${pdfPath}`);
      return null;
    }

    return {
      orderNumber,
      orderDate,
      totalAmount,
      fileName: pdfPath,
    };
  } catch (error) {
    console.error(`Error parsing PDF ${pdfPath}:`, error);
    return null;
  }
}

/**
 * Extract Amazon order number from PDF text
 * Patterns: 123-4567890-1234567
 */
function extractOrderNumber(text: string): string | null {
  const patterns = [
    /注文番号[：:\s]*(\d{3}-\d{7}-\d{7})/,
    /Order Number[：:\s]*(\d{3}-\d{7}-\d{7})/i,
    /(\d{3}-\d{7}-\d{7})/,
  ];

  for (const pattern of patterns) {
    const match = text.match(pattern);
    if (match && match[1]) {
      return match[1];
    }
  }

  return null;
}

/**
 * Extract order date from PDF text
 * Converts to YYYY-MM-DD format
 */
function extractOrderDate(text: string): string | null {
  const patterns = [
    // Japanese format: 2025年2月15日
    /注文日[：:\s]*(\d{4})年(\d{1,2})月(\d{1,2})日/,
    // English format: February 15, 2025
    /Order Date[：:\s]*([A-Za-z]+)\s+(\d{1,2}),?\s+(\d{4})/i,
    // ISO format: 2025-02-15
    /(\d{4})-(\d{2})-(\d{2})/,
  ];

  for (const pattern of patterns) {
    const match = text.match(pattern);
    if (match) {
      // Japanese format
      if (pattern === patterns[0] && match[1] && match[2] && match[3]) {
        const year = match[1];
        const month = match[2].padStart(2, '0');
        const day = match[3].padStart(2, '0');
        return `${year}-${month}-${day}`;
      }

      // English format
      if (pattern === patterns[1] && match[1] && match[2] && match[3]) {
        const monthName = match[1];
        const day = match[2].padStart(2, '0');
        const year = match[3];
        const month = parseEnglishMonth(monthName);
        if (month) {
          return `${year}-${month}-${day}`;
        }
      }

      // ISO format
      if (pattern === patterns[2] && match[1] && match[2] && match[3]) {
        return `${match[1]}-${match[2]}-${match[3]}`;
      }
    }
  }

  return null;
}

/**
 * Extract total amount from PDF text
 */
function extractTotalAmount(text: string): number | null {
  const patterns = [
    // Japanese: 合計: ¥12,980 or 合計：12,980円
    /合計[：:\s]*[¥￥]?([\d,]+)円?/,
    /総額[：:\s]*[¥￥]?([\d,]+)円?/,
    // English: Total: ¥12,980 or Total: $129.80
    /Total[：:\s]*[¥￥$]?([\d,]+\.?\d*)/i,
    /Grand Total[：:\s]*[¥￥$]?([\d,]+\.?\d*)/i,
  ];

  for (const pattern of patterns) {
    const match = text.match(pattern);
    if (match && match[1]) {
      const amountStr = match[1].replace(/,/g, '');
      const amount = parseInt(amountStr, 10);
      if (!isNaN(amount)) {
        return amount;
      }
    }
  }

  return null;
}

/**
 * Convert English month name to zero-padded month number
 */
function parseEnglishMonth(monthName: string): string | null {
  const months: { [key: string]: string } = {
    january: '01',
    february: '02',
    march: '03',
    april: '04',
    may: '05',
    june: '06',
    july: '07',
    august: '08',
    september: '09',
    october: '10',
    november: '11',
    december: '12',
  };

  const normalized = monthName.toLowerCase();
  for (const [name, num] of Object.entries(months)) {
    if (name.startsWith(normalized) || normalized.startsWith(name.substring(0, 3))) {
      return num;
    }
  }

  return null;
}
