export interface PocketPrompt {
  ID: string;
  Name: string;
  Summary: string;
  Content: string;
  Tags: string[];
  Version: string;
  TemplateRef?: string;
  Pack?: string;
  UpdatedAt: string;
  CreatedAt: string;
  FilePath: string;
  ContentHash: string;
  Metadata?: any;
}

export interface PocketPromptTemplate {
  id: string;
  name: string;
  description: string;
  content: string;
  version: string;
  slots?: TemplateSlot[];
}

export interface TemplateSlot {
  name: string;
  description: string;
  required: boolean;
  default?: string;
}

export interface SearchResult {
  prompts: PocketPrompt[];
  total: number;
}

export interface ServerStatus {
  status: string;
  service: string;
}

export interface BooleanExpression {
  type: "tag" | "and" | "or" | "not";
  value: string | BooleanExpression[];
}

export interface SavedSearch {
  name: string;
  description?: string;
  expression: BooleanExpression;
  text_query?: string;
  created_at: string;
  updated_at: string;
}

export interface BooleanSearchMode {
  type: "boolean";
  expression: string;
}

export interface SavedSearchMode {
  type: "saved";
  searchName: string;
}

export interface FuzzySearchMode {
  type: "fuzzy";
  query: string;
}

export type SearchMode = BooleanSearchMode | SavedSearchMode | FuzzySearchMode;
