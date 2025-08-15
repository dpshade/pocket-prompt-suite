// @ts-nocheck
import {
  ActionPanel,
  Action,
  List,
  showToast,
  Toast,
  Clipboard,
  Icon,
  Color,
  Form,
  useNavigation,
} from "@raycast/api";
import { useState, useEffect, useMemo } from "react";
import {
  useUnifiedSearch,
  useServerHealth,
  useTags,
  useSavedSearches,
} from "./hooks/usePocketPrompt";
import { useCachedPromise } from "@raycast/utils";
import { PocketPrompt, SearchMode } from "./types";
import { pocketPromptAPI } from "./utils/api";
import PromptDetailView from "./components/PromptDetailView";
import AddPrompt from "./add-prompt";

interface SearchPromptsProps {
  initialSearchMode?: SearchMode;
}

export default function SearchPrompts({ initialSearchMode }: SearchPromptsProps = {}) {
  const [searchText, setSearchText] = useState(() => {
    // Initialize search text if provided
    if (initialSearchMode?.type === "fuzzy") {
      return initialSearchMode.query;
    } else if (initialSearchMode?.type === "boolean") {
      return initialSearchMode.expression;
    }
    return "";
  });
  const [selectedFilter, setSelectedFilter] = useState<string>("");
  const [selectedPacks, setSelectedPacks] = useState<string[]>(["personal"]);

  const {
    data: serverHealth,
    isLoading: healthLoading,
    error: healthError,
  } = useServerHealth();
  const { data: tags } = useTags();
  const { data: savedSearches } = useSavedSearches();
  const { data: availablePacks } = useCachedPromise(
    async () => pocketPromptAPI.getAvailablePacks(),
    [],
    { initialData: { "Personal Library (default)": "personal" } }
  );

  // Initialize search mode when data is loaded
  useEffect(() => {
    if (initialSearchMode && selectedFilter === "") {
      if (initialSearchMode.type === "saved" && savedSearches && savedSearches.length > 0) {
        const searchExists = savedSearches.includes(initialSearchMode.searchName);
        if (searchExists) {
          setSelectedFilter(`saved:${initialSearchMode.searchName}`);
        }
      }
      // For boolean searches, we could set a tag filter if it matches exactly
      // For fuzzy searches, we just rely on the searchText initialization
    }
  }, [initialSearchMode, savedSearches, selectedFilter]);

  // Handle pack selection separately to avoid re-render loops
  useEffect(() => {
    if (selectedFilter.startsWith("pack:")) {
      const packName = selectedFilter.replace("pack:", "");
      const newPacks = selectedPacks.includes(packName) 
        ? selectedPacks.filter(p => p !== packName) // Remove if already selected
        : [...selectedPacks, packName]; // Add if not selected
      
      if (JSON.stringify(newPacks) !== JSON.stringify(selectedPacks)) {
        setSelectedPacks(newPacks);
      }
      // Clear the filter to allow multiple selections
      setSelectedFilter("");
    }
  }, [selectedFilter, selectedPacks]);

  // Parse bracket syntax and create search analysis
  const searchAnalysis = useMemo(() => {
    if (selectedFilter.startsWith("saved:")) {
      return {
        type: "saved" as const,
        fuzzyQuery: searchText,
        booleanExpr: "",
        searchName: selectedFilter.replace("saved:", ""),
      };
    }
    if (selectedFilter.startsWith("tag:")) {
      const tag = selectedFilter.replace("tag:", "");
      return {
        type: "boolean" as const,
        fuzzyQuery: "",
        booleanExpr: tag,
      };
    }

    // Parse bracket syntax: text [boolean expr] more text
    const bracketPattern = /\[([^\[\]]+)\]/g;
    const matches = [...searchText.matchAll(bracketPattern)];

    if (matches.length > 0) {
      // Extract boolean expressions from brackets
      const booleanParts = matches.map((match) => match[1].trim());
      const booleanExpr = booleanParts.join(" AND ");

      // Remove brackets to get fuzzy part
      const fuzzyQuery = searchText.replace(bracketPattern, "").trim();

      return {
        type:
          fuzzyQuery && booleanExpr
            ? ("hybrid" as const)
            : booleanExpr
              ? ("boolean" as const)
              : ("fuzzy" as const),
        fuzzyQuery,
        booleanExpr,
      };
    }

    // No brackets - pure fuzzy search
    return {
      type: "fuzzy" as const,
      fuzzyQuery: searchText,
      booleanExpr: "",
    };
  }, [searchText, selectedFilter]);

  const {
    data: prompts,
    isLoading,
    error,
    revalidate,
  } = useUnifiedSearch(
    searchAnalysis.fuzzyQuery,
    searchAnalysis.booleanExpr,
    searchAnalysis.type,
    searchAnalysis.searchName,
    selectedPacks,
  );

  useEffect(() => {
    if (healthError) {
      showToast({
        style: Toast.Style.Failure,
        title: "Server Connection Failed",
        message: "Make sure Pocket Prompt server is running on localhost:8080",
      });
    }
  }, [healthError]);

  const copyPromptToClipboard = async (prompt: PocketPrompt) => {
    try {
      await Clipboard.copy(prompt.Content);
      showToast({
        style: Toast.Style.Success,
        title: "Copied to Clipboard",
        message: prompt.Name,
      });
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Failed to Copy",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const getAccessories = (prompt: PocketPrompt) => {
    const accessories = [];

    if (prompt.Tags && prompt.Tags.length > 0) {
      accessories.push({
        text: prompt.Tags.slice(0, 2).join(", "),
        icon: Icon.Tag,
      });
    }

    return accessories;
  };

  const getSearchPlaceholder = () => {
    if (selectedFilter.startsWith("saved:")) {
      const searchName = selectedFilter.replace("saved:", "");
      return `Executing saved search: ${searchName}`;
    }
    if (selectedFilter.startsWith("tag:")) {
      const tag = selectedFilter.replace("tag:", "");
      return `Filtering by tag: ${tag}`;
    }
    if (selectedPacks.length === 1 && selectedPacks[0] !== "personal") {
      return `Search ${selectedPacks[0]} pack...`;
    }
    if (selectedPacks.length > 1) {
      return `Search ${selectedPacks.length} selected packs...`;
    }
    return "Search prompts...";
  };

  const getEmptyViewContent = () => {
    if (selectedFilter.startsWith("saved:")) {
      const searchName = selectedFilter.replace("saved:", "");
      return {
        title: "No Results",
        description: `Saved search "${searchName}" returned no results`,
        icon: Icon.Bookmark,
      };
    }

    if (selectedFilter.startsWith("tag:")) {
      const tag = selectedFilter.replace("tag:", "");
      return {
        title: "No Prompts",
        description: `No prompts found with tag "${tag}"`,
        icon: Icon.Tag,
      };
    }

    if (!searchText.trim()) {
      const packContext = selectedPacks.length === 1 && selectedPacks[0] !== "personal"
        ? `Searching in ${selectedPacks[0]} pack.\n\n`
        : selectedPacks.length > 1
        ? `Searching in ${selectedPacks.length} selected packs.\n\n`
        : "";
        
      return {
        title: "Search Your Prompts",
        description:
          packContext +
          `Start typing to search, or use the filter dropdown.\n\n` +
          `• Fuzzy search: "machine learning"\n` +
          `• Boolean search: "[ai AND agent]"\n` +
          `• Mixed search: "tutorial [python OR javascript]"\n\n` +
          `Use the dropdown to select packs (multi-select supported).`,
        icon: Icon.MagnifyingGlass,
      };
    }

    if (searchAnalysis.type === "boolean") {
      return {
        title: "No Results",
        description: `Boolean search "${searchAnalysis.booleanExpr}" returned no results`,
        icon: Icon.Code,
      };
    }

    if (searchAnalysis.type === "hybrid") {
      return {
        title: "No Results",
        description: `Hybrid search "${searchAnalysis.fuzzyQuery}" + [${searchAnalysis.booleanExpr}] returned no results`,
        icon: Icon.Code,
      };
    }

    return {
      title: "No Results",
      description: `No prompts match "${searchText}"`,
      icon: Icon.ExclamationMark,
    };
  };

  const emptyViewContent = getEmptyViewContent();

  return (
    <List
      isLoading={isLoading || healthLoading}
      onSearchTextChange={setSearchText}
      searchBarPlaceholder={getSearchPlaceholder()}
      throttle={true}
      searchBarAccessory={
        <List.Dropdown
          tooltip="Filter and Search Options"
          placeholder={selectedPacks.length === 1 && selectedPacks[0] === "personal" 
            ? "Personal Library" 
            : selectedPacks.length === 1 
            ? selectedPacks[0] 
            : `${selectedPacks.length} Packs Selected`}
          value={selectedFilter}
          onChange={(value) => setSelectedFilter(value || "")}
        >
          <List.Dropdown.Section title="Packs (Multi-select)">
            {availablePacks && Object.entries(availablePacks).map(([displayName, packName]) => {
              const isSelected = selectedPacks.includes(packName);
              return (
                <List.Dropdown.Item
                  key={`pack:${packName}`}
                  title={isSelected ? `✓ ${displayName}` : displayName}
                  value={`pack:${packName}`}
                  icon={{ 
                    source: isSelected ? Icon.CheckCircle : Icon.Box, 
                    tintColor: isSelected ? Color.Green : Color.Orange 
                  }}
                />
              );
            })}
          </List.Dropdown.Section>


          {(savedSearches || []).length > 0 && (
            <List.Dropdown.Section title="Saved Searches">
              {(savedSearches || []).map((searchName) => (
                <List.Dropdown.Item
                  key={`saved:${searchName}`}
                  title={searchName}
                  value={`saved:${searchName}`}
                  icon={{ source: Icon.Bookmark, tintColor: Color.Purple }}
                />
              ))}
            </List.Dropdown.Section>
          )}

          {(tags || []).length > 0 && (
            <List.Dropdown.Section title="Tags">
              {(tags || []).map((tag) => (
                <List.Dropdown.Item
                  key={`tag:${tag}`}
                  title={tag}
                  value={`tag:${tag}`}
                  icon={{ source: Icon.Tag, tintColor: Color.Blue }}
                />
              ))}
            </List.Dropdown.Section>
          )}
        </List.Dropdown>
      }
    >
      {healthError ? (
        <List.EmptyView
          icon={Icon.Warning}
          title="Server Not Available"
          description="Make sure Pocket Prompt server is running on localhost:8080"
          actions={
            <ActionPanel>
              <Action
                title="Retry Connection"
                icon={Icon.RotateClockwise}
                onAction={() => revalidate()}
              />
              <Action.Push
                title="New Prompt"
                icon={Icon.Plus}
                target={<AddPrompt onRefresh={revalidate} />}
                shortcut={{ modifiers: ["cmd"], key: "n" }}
              />
            </ActionPanel>
          }
        />
      ) : (prompts || []).length === 0 ? (
        <List.EmptyView
          icon={emptyViewContent.icon}
          title={emptyViewContent.title}
          description={emptyViewContent.description}
          actions={
            <ActionPanel>
              <Action.Push
                title="New Prompt"
                icon={Icon.Plus}
                target={<AddPrompt onRefresh={revalidate} />}
                shortcut={{ modifiers: ["cmd"], key: "n" }}
              />
              <Action
                title="Refresh"
                icon={Icon.RotateClockwise}
                onAction={() => revalidate()}
              />
              <Action
                title="Clear Filters"
                icon={Icon.Trash}
                onAction={() => {
                  setSelectedFilter("");
                  setSearchText("");
                  setSelectedPacks(["personal"]);
                }}
              />
            </ActionPanel>
          }
        />
      ) : (
        (prompts || []).map((prompt) => (
          <List.Item
            key={prompt.ID}
            title={prompt.Name}
            subtitle={prompt.Summary}
            accessories={[
              ...getAccessories(prompt),
              ...(searchAnalysis.type === "boolean"
                ? [{ text: "Boolean", icon: Icon.Code }]
                : searchAnalysis.type === "hybrid"
                  ? [{ text: "Hybrid", icon: Icon.Code }]
                  : []),
            ]}
            actions={
              <ActionPanel>
                <ActionPanel.Section title="Prompt Actions">
                  <Action
                    title="Copy to Clipboard"
                    icon={Icon.Clipboard}
                    onAction={() => copyPromptToClipboard(prompt)}
                  />
                  <Action.Push
                    title="Show Details"
                    icon={Icon.Eye}
                    target={
                      <PromptDetailView
                        prompt={prompt}
                        onRefresh={revalidate}
                      />
                    }
                    shortcut={{ modifiers: ["shift"], key: "enter" }}
                  />
                  <Action
                    title="Copy Raw Content"
                    icon={Icon.Document}
                    onAction={async () => {
                      await Clipboard.copy(prompt.Content);
                      showToast({
                        style: Toast.Style.Success,
                        title: "Copied Raw Content",
                        message: prompt.Name,
                      });
                    }}
                  />
                </ActionPanel.Section>
                <ActionPanel.Section title="Search Actions">
                  <Action
                    title="Clear Filters"
                    icon={Icon.Trash}
                    onAction={() => {
                      setSelectedFilter("");
                      setSearchText("");
                      setSelectedPacks(["personal"]);
                    }}
                    shortcut={{ modifiers: ["cmd", "shift"], key: "k" }}
                  />
                </ActionPanel.Section>
                <ActionPanel.Section title="Navigation">
                  <Action.Push
                    title="New Prompt"
                    icon={Icon.Plus}
                    target={<AddPrompt onRefresh={revalidate} />}
                    shortcut={{ modifiers: ["cmd"], key: "n" }}
                  />
                  <Action
                    title="Refresh"
                    icon={Icon.RotateClockwise}
                    shortcut={{ modifiers: ["cmd"], key: "r" }}
                    onAction={() => revalidate()}
                  />
                </ActionPanel.Section>
              </ActionPanel>
            }
          />
        ))
      )}
    </List>
  );
}
