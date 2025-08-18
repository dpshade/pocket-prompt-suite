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
        // If no query, return prompts based on pack selection
        if (packs && packs.length > 0) {
          // Handle multiple pack selection
          if (packs.length === 1 && packs[0] === "personal") {
            // Just personal library
            return pocketPromptAPI.listAllPrompts();
          }
          
          // Multiple packs or non-personal packs
          const allPackPrompts = await Promise.all(
            packs.map(pack =>
              pack === "personal"
                ? pocketPromptAPI.listAllPrompts()
                : pocketPromptAPI.listPromptsByPack(pack)
            )
          );
          const combined = allPackPrompts.flat();
          const seen = new Set<string>();
          return combined.filter(prompt => {
            if (seen.has(prompt.ID)) return false;
            seen.add(prompt.ID);
            return true;
          });
        }
        // Default to personal library if no packs specified
        return pocketPromptAPI.listAllPrompts();
      }

      switch (type) {
        case "saved":
          return savedSearchName
            ? pocketPromptAPI.executeSavedSearch(savedSearchName)
            : [];
        case "boolean":
          return pocketPromptAPI.booleanSearch(expr, packs);
        case "fuzzy":
          return pocketPromptAPI.searchPrompts(fuzzy, packs);
        case "hybrid":
          return pocketPromptAPI.hybridSearch(fuzzy, expr, packs);
        default:
          // Default to hybrid search with pack filtering
          return pocketPromptAPI.hybridSearch(fuzzy, expr, packs);
      }
    },
    [fuzzyQuery, booleanExpr, searchType, searchName, packNames],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}
