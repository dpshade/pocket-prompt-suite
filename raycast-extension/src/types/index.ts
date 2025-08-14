export interface PocketPrompt {
  ID: string;
  Name: string;
  Summary: string;
  Content: string;
  Tags: string[];
  Version: string;
  TemplateRef?: string;
  Variables?: PromptVariable[];
  UpdatedAt: string;
  CreatedAt: string;
  FilePath: string;
  ContentHash: string;
  Metadata?: any;
}

export interface PromptVariable {
  name: string;
  type: "string" | "number" | "boolean" | "list";
  required: boolean;
  default?: string | number | boolean;
  description?: string;
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

export interface RenderParams {
  [key: string]: string | number | boolean;
}

export interface SavedSearch {
  name: string;
  expression: string;
}

export interface BooleanSearchMode {
  type: 'boolean';
  expression: string;
}

export interface SavedSearchMode {
  type: 'saved';
  searchName: string;
}

export interface FuzzySearchMode {
  type: 'fuzzy';
  query: string;
}

export type SearchMode = BooleanSearchMode | SavedSearchMode | FuzzySearchMode;