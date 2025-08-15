import { useCachedPromise } from "@raycast/utils";
import { pocketPromptAPI } from "../utils/api";
import { PocketPrompt, ServerStatus } from "../types";

export function useServerHealth() {
  return useCachedPromise(
    async (): Promise<ServerStatus> => {
      return pocketPromptAPI.checkHealth();
    },
    [],
    {
      keepPreviousData: true,
      initialData: undefined,
    },
  );
}

export function usePrompts() {
  return useCachedPromise(
    async (): Promise<PocketPrompt[]> => {
      return pocketPromptAPI.listAllPrompts();
    },
    [],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useSearchPrompts(query: string) {
  return useCachedPromise(
    async (searchQuery: string): Promise<PocketPrompt[]> => {
      if (!searchQuery.trim()) {
        return pocketPromptAPI.listAllPrompts();
      }
      return pocketPromptAPI.searchPrompts(searchQuery);
    },
    [query],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useTags() {
  return useCachedPromise(
    async (): Promise<string[]> => {
      return pocketPromptAPI.getTags();
    },
    [],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function usePromptsByTag(tag: string | null) {
  return useCachedPromise(
    async (selectedTag: string): Promise<PocketPrompt[]> => {
      return pocketPromptAPI.getPromptsByTag(selectedTag);
    },
    [tag!],
    {
      execute: !!tag,
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useBooleanSearch(expression: string) {
  return useCachedPromise(
    async (booleanExpression: string): Promise<PocketPrompt[]> => {
      return pocketPromptAPI.booleanSearch(booleanExpression);
    },
    [expression],
    {
      execute: !!expression.trim(),
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useSavedSearches() {
  return useCachedPromise(
    async (): Promise<string[]> => {
      return pocketPromptAPI.listSavedSearches();
    },
    [],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useSavedSearch(searchName: string | null) {
  return useCachedPromise(
    async (name: string): Promise<PocketPrompt[]> => {
      return pocketPromptAPI.executeSavedSearch(name);
    },
    [searchName!],
    {
      execute: !!searchName,
      keepPreviousData: true,
      initialData: [],
    },
  );
}

export function useUnifiedSearch(
  fuzzyQuery: string,
  booleanExpr: string,
  searchType: "fuzzy" | "boolean" | "hybrid" | "saved",
  searchName?: string,
  packNames?: string[],
) {
  return useCachedPromise(
    async (
      fuzzy: string,
      expr: string,
      type: string,
      savedSearchName?: string,
      packs?: string[],
    ): Promise<PocketPrompt[]> => {
      if (!fuzzy.trim() && !expr.trim() && type !== "saved") {
        // If no search query, return all prompts from the selected packs
        if (packs && packs.length > 0) {
          // Fetch prompts from multiple packs and combine them
          const allPackPrompts = await Promise.all(
            packs.map(pack => 
              pack === "personal" 
                ? pocketPromptAPI.listAllPrompts() 
                : pocketPromptAPI.listPromptsByPack(pack)
            )
          );
          // Flatten the arrays and deduplicate by ID
          const combined = allPackPrompts.flat();
          const seen = new Set<string>();
          return combined.filter(prompt => {
            if (seen.has(prompt.ID)) return false;
            seen.add(prompt.ID);
            return true;
          });
        } else {
          return pocketPromptAPI.listAllPrompts();
        }
      }

      switch (type) {
        case "saved":
          return savedSearchName
            ? pocketPromptAPI.executeSavedSearch(savedSearchName)
            : [];
        case "boolean":
        case "fuzzy":
        case "hybrid":
        default:
          // Use new API method that sends separate parameters
          return pocketPromptAPI.hybridSearch(fuzzy, expr);
      }
    },
    [fuzzyQuery, booleanExpr, searchType, searchName, packNames],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}
