export interface SearchAnalysis {
  type: "fuzzy" | "boolean";
  query: string;
  confidence: number;
}

export function analyzeSearchQuery(query: string): SearchAnalysis {
  const trimmedQuery = query.trim();

  if (!trimmedQuery) {
    return { type: "fuzzy", query: trimmedQuery, confidence: 1.0 };
  }

  // Boolean operators detection
  const booleanOperators = /\b(AND|OR|NOT)\b/i;
  const hasParentheses = /[()]/;
  const hasQuotes = /['"]/;

  // Strong boolean indicators
  if (booleanOperators.test(trimmedQuery)) {
    return { type: "boolean", query: trimmedQuery, confidence: 0.9 };
  }

  if (hasParentheses.test(trimmedQuery)) {
    return { type: "boolean", query: trimmedQuery, confidence: 0.8 };
  }

  // Weak boolean indicators (could be either)
  if (hasQuotes.test(trimmedQuery)) {
    return { type: "boolean", query: trimmedQuery, confidence: 0.6 };
  }

  // Check for tag-like patterns (common in boolean searches)
  const tagPattern = /^[\w-]+(\s+(AND|OR|NOT)\s+[\w-]+)*$/i;
  if (tagPattern.test(trimmedQuery)) {
    return { type: "boolean", query: trimmedQuery, confidence: 0.7 };
  }

  // Default to fuzzy search
  return { type: "fuzzy", query: trimmedQuery, confidence: 0.8 };
}

export function formatBooleanExpression(query: string): string {
  // Auto-correct common patterns
  return query
    .replace(/\s+and\s+/gi, " AND ")
    .replace(/\s+or\s+/gi, " OR ")
    .replace(/\s+not\s+/gi, " NOT ")
    .trim();
}
