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
  query: string,
  searchType: "fuzzy" | "boolean" | "saved",
  searchName?: string,
) {
  return useCachedPromise(
    async (
      searchQuery: string,
      type: string,
      savedSearchName?: string,
    ): Promise<PocketPrompt[]> => {
      if (!searchQuery.trim() && type !== "saved") {
        return pocketPromptAPI.listAllPrompts();
      }

      switch (type) {
        case "boolean":
          return pocketPromptAPI.booleanSearch(searchQuery);
        case "saved":
          return savedSearchName
            ? pocketPromptAPI.executeSavedSearch(savedSearchName)
            : [];
        case "fuzzy":
        default:
          return pocketPromptAPI.searchPrompts(searchQuery);
      }
    },
    [query, searchType, searchName],
    {
      keepPreviousData: true,
      initialData: [],
    },
  );
}
