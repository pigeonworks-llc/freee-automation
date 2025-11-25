/**
 * Deal Matcher
 * Matches Amazon receipts with freee deals based on date and amount
 */

import dayjs from 'dayjs';
import { Deal } from './types';
import { AmazonReceiptData } from './pdf-parser';

export interface MatchResult {
  status: 'unique' | 'multiple' | 'none';
  matches: Deal[];
}

/**
 * Find matching deals for an Amazon receipt
 * Matching criteria:
 * - Date within ±3 days
 * - Amount exactly matches
 */
export function findMatchingDeals(
  receipt: AmazonReceiptData,
  deals: Deal[]
): MatchResult {
  const matches: Deal[] = [];

  for (const deal of deals) {
    if (matchDeal(deal, receipt)) {
      matches.push(deal);
    }
  }

  if (matches.length === 0) {
    return { status: 'none', matches: [] };
  } else if (matches.length === 1) {
    return { status: 'unique', matches };
  } else {
    return { status: 'multiple', matches };
  }
}

/**
 * Check if a deal matches the receipt criteria
 */
function matchDeal(deal: Deal, receipt: AmazonReceiptData): boolean {
  // Condition 1: Date within ±3 days
  const dateDiff = Math.abs(
    dayjs(deal.issue_date).diff(dayjs(receipt.orderDate), 'day')
  );
  if (dateDiff > 3) {
    return false;
  }

  // Condition 2: Amount exactly matches
  if (deal.amount !== receipt.totalAmount) {
    return false;
  }

  return true;
}

/**
 * Filter deals by date range
 * Returns deals within ±1 month of the given date
 */
export function filterDealsByDateRange(
  deals: Deal[],
  targetDate: string
): Deal[] {
  const start = dayjs(targetDate).subtract(1, 'month');
  const end = dayjs(targetDate).add(1, 'month');

  return deals.filter((deal) => {
    const issueDate = dayjs(deal.issue_date);
    return issueDate.isAfter(start) && issueDate.isBefore(end);
  });
}
