/**
 * Account Item Mapping Rules
 * Maps transaction descriptions to freee account items
 */

export interface AccountRule {
  pattern: RegExp;
  accountName: string;  // 勘定科目名 (will be resolved to ID)
  taxCode: number;      // 税区分コード
  description?: string; // Optional override for deal description
}

/**
 * Rule definitions for automatic account classification
 *
 * Tax codes:
 * - 136: 課税仕入10%(税込)
 * - 137: 課税仕入8%(軽減・税込)
 * - 21: 課税売上10%
 */
export const accountRules: AccountRule[] = [
  // Books & Subscriptions (新聞図書費)
  {
    pattern: /amazon|アマゾン|amzn/i,
    accountName: '新聞図書費',
    taxCode: 136,
  },
  {
    pattern: /audible/i,
    accountName: '新聞図書費',
    taxCode: 136,
  },
  {
    pattern: /kindle/i,
    accountName: '新聞図書費',
    taxCode: 136,
  },

  // Training & Education (研修費)
  {
    pattern: /teachable|school|udemy|coursera/i,
    accountName: '研修費',
    taxCode: 136,
  },
  {
    pattern: /nomad\.?love|nextraveler/i,
    accountName: '研修費',
    taxCode: 136,
  },

  // Software & Services (通信費 or 支払手数料)
  {
    pattern: /github|openai|anthropic|cursor/i,
    accountName: '支払手数料',
    taxCode: 136,
  },
  {
    pattern: /apple.*itunes|app\s*store/i,
    accountName: '通信費',
    taxCode: 136,
  },

  // Cloud & Hosting (通信費)
  {
    pattern: /aws|google\s*cloud|azure|vercel|netlify/i,
    accountName: '通信費',
    taxCode: 136,
  },
];

/**
 * Find matching rule for a transaction description
 */
export function findMatchingRule(description: string): AccountRule | null {
  for (const rule of accountRules) {
    if (rule.pattern.test(description)) {
      return rule;
    }
  }
  return null;
}

/**
 * Account item cache (name -> id mapping)
 */
export interface AccountItemCache {
  [name: string]: number;
}
