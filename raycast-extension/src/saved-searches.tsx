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
  Alert,
  confirmAlert,
} from "@raycast/api";
import { useState, useEffect } from "react";
import { useCachedPromise } from "@raycast/utils";
import { SavedSearch, BooleanExpression, SearchMode } from "./types";
import { pocketPromptAPI } from "./utils/api";
import { useTags } from "./hooks/usePocketPrompt";

// Helper functions for boolean expression handling
const formatBooleanExpression = (expr: BooleanExpression): string => {
  switch (expr.type) {
    case "tag":
      return expr.value as string;
    case "and":
      return (expr.value as BooleanExpression[])
        .map(e => formatBooleanExpression(e))
        .join(" AND ");
    case "or":
      return (expr.value as BooleanExpression[])
        .map(e => formatBooleanExpression(e))
        .join(" OR ");
    case "not":
      const negated = (expr.value as BooleanExpression[])[0];
      return `NOT ${formatBooleanExpression(negated)}`;
    default:
      return "";
  }
};

const parseBooleanExpression = (input: string): BooleanExpression => {
  const trimmed = input.trim();
  
  // Handle NOT operations first
  if (trimmed.toUpperCase().startsWith("NOT ")) {
    const inner = trimmed.substring(4).trim();
    return {
      type: "not",
      value: [parseBooleanExpression(inner)]
    };
  }
  
  // Split by OR (lower precedence)
  const orParts = trimmed.split(/\s+OR\s+/i);
  if (orParts.length > 1) {
    return {
      type: "or",
      value: orParts.map(part => parseBooleanExpression(part.trim()))
    };
  }
  
  // Split by AND (higher precedence)
  const andParts = trimmed.split(/\s+AND\s+/i);
  if (andParts.length > 1) {
    return {
      type: "and",
      value: andParts.map(part => parseBooleanExpression(part.trim()))
    };
  }
  
  // Remove parentheses if present
  if (trimmed.startsWith("(") && trimmed.endsWith(")")) {
    return parseBooleanExpression(trimmed.slice(1, -1));
  }
  
  // Single tag expression
  return {
    type: "tag",
    value: trimmed
  };
};

export default function SavedSearches() {
  const [isLoading, setIsLoading] = useState(false);
  const { push } = useNavigation();

  const {
    data: savedSearches,
    isLoading: searchesLoading,
    error: searchesError,
    revalidate,
  } = useCachedPromise(
    async () => pocketPromptAPI.listSavedSearchesDetailed(),
    [],
    { initialData: [] }
  );

  useEffect(() => {
    if (searchesError) {
      showToast({
        style: Toast.Style.Failure,
        title: "Failed to Load Saved Searches",
        message: "Make sure Pocket Prompt server is running on localhost:8080",
      });
    }
  }, [searchesError]);

  const executeSavedSearch = async (search: SavedSearch) => {
    try {
      // Import the search-prompts component and push with saved search pre-applied
      const SearchPromptsComponent = (await import("./search-prompts")).default;
      
      push(
        <SearchPromptsComponent 
          initialSearchMode={{
            type: "saved",
            searchName: search.name
          } as const}
        />
      );
      
      showToast({
        style: Toast.Style.Success,
        title: "Opening Search",
        message: `Loading "${search.name}" search results`,
      });
      
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Search Failed",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const deleteSavedSearch = async (search: SavedSearch) => {
    const confirmed = await confirmAlert({
      title: "Delete Saved Search",
      message: `Are you sure you want to delete "${search.name}"?`,
      primaryAction: {
        title: "Delete",
        style: Alert.ActionStyle.Destructive,
      },
    });

    if (!confirmed) return;

    try {
      await pocketPromptAPI.deleteSavedSearch(search.name);
      showToast({
        style: Toast.Style.Success,
        title: "Search Deleted",
        message: `Deleted "${search.name}"`,
      });
      revalidate();
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Delete Failed",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const copySearchExpression = async (search: SavedSearch) => {
    try {
      const expressionString = formatBooleanExpression(search.expression);
      await Clipboard.copy(expressionString);
      showToast({
        style: Toast.Style.Success,
        title: "Copied to Clipboard",
        message: expressionString,
      });
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Copy Failed",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };


  const getAccessories = (search: SavedSearch) => {
    const accessories = [];
    
    const expressionText = formatBooleanExpression(search.expression);
    if (expressionText) {
      accessories.push({
        text: expressionText,
        icon: Icon.Code,
      });
    }

    if (search.text_query) {
      accessories.push({
        text: `Text: ${search.text_query}`,
        icon: Icon.MagnifyingGlass,
      });
    }

    return accessories;
  };

  return (
    <List
      isLoading={searchesLoading || isLoading}
      searchBarPlaceholder="Search saved searches..."
    >
      {searchesError ? (
        <List.EmptyView
          icon={Icon.Warning}
          title="Server Not Available"
          description="Make sure Pocket Prompt server is running on localhost:8080"
          actions={
            <ActionPanel>
              <Action
                title="Retry"
                icon={Icon.RotateClockwise}
                onAction={() => revalidate()}
              />
              <Action.Push
                title="Create New Search"
                icon={Icon.Plus}
                target={<CreateSavedSearchForm onRefresh={revalidate} />}
                shortcut={{ modifiers: ["cmd"], key: "n" }}
              />
            </ActionPanel>
          }
        />
      ) : (savedSearches || []).length === 0 ? (
        <List.EmptyView
          icon={Icon.Bookmark}
          title="No Saved Searches"
          description="Create your first saved search to get started"
          actions={
            <ActionPanel>
              <Action.Push
                title="Create New Search"
                icon={Icon.Plus}
                target={<CreateSavedSearchForm onRefresh={revalidate} />}
                shortcut={{ modifiers: ["cmd"], key: "n" }}
              />
            </ActionPanel>
          }
        />
      ) : (
        (savedSearches || []).map((search) => (
          <List.Item
            key={search.name}
            title={search.name}
            subtitle={search.description || ""}
            accessories={getAccessories(search)}
            actions={
              <ActionPanel>
                <ActionPanel.Section title="Search Actions">
                  <Action
                    title="Execute Search"
                    icon={Icon.Play}
                    onAction={() => executeSavedSearch(search)}
                  />
                  <Action
                    title="Copy Expression"
                    icon={Icon.Clipboard}
                    onAction={() => copySearchExpression(search)}
                  />
                </ActionPanel.Section>
                <ActionPanel.Section title="Management">
                  <Action.Push
                    title="Edit Search"
                    icon={Icon.Pencil}
                    target={<EditSavedSearchForm search={search} onRefresh={revalidate} />}
                    shortcut={{ modifiers: ["cmd"], key: "e" }}
                  />
                  <Action
                    title="Delete Search"
                    icon={Icon.Trash}
                    style={Action.Style.Destructive}
                    onAction={() => deleteSavedSearch(search)}
                    shortcut={{ modifiers: ["cmd"], key: "d" }}
                  />
                </ActionPanel.Section>
                <ActionPanel.Section title="Navigation">
                  <Action.Push
                    title="Create New Search"
                    icon={Icon.Plus}
                    target={<CreateSavedSearchForm onRefresh={revalidate} />}
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

interface CreateSavedSearchFormProps {
  onRefresh: () => void;
}

function CreateSavedSearchForm({ onRefresh }: CreateSavedSearchFormProps) {
  const { pop } = useNavigation();
  const [nameValue, setNameValue] = useState("");
  const [expressionValue, setExpressionValue] = useState("");
  const [textQueryValue, setTextQueryValue] = useState("");
  const [expressionError, setExpressionError] = useState<string>("");
  
  const { data: tags } = useTags();

  const validateExpression = (input: string) => {
    if (!input.trim()) {
      setExpressionError("");
      return;
    }
    
    try {
      parseBooleanExpression(input);
      setExpressionError("");
    } catch (error) {
      setExpressionError("Invalid boolean expression");
    }
  };

  const handleExpressionChange = (value: string) => {
    setExpressionValue(value);
    validateExpression(value);
  };


  const handleSubmit = async () => {
    if (!nameValue.trim() || !expressionValue.trim()) {
      showToast({
        style: Toast.Style.Failure,
        title: "Validation Error",
        message: "Name and expression are required",
      });
      return;
    }

    if (expressionError) {
      showToast({
        style: Toast.Style.Failure,
        title: "Expression Error",
        message: expressionError,
      });
      return;
    }

    try {
      const expression = parseBooleanExpression(expressionValue);

      await pocketPromptAPI.createSavedSearch({
        name: nameValue.trim(),
        expression,
        textQuery: textQueryValue.trim() || undefined,
      });

      showToast({
        style: Toast.Style.Success,
        title: "Search Created",
        message: `Created "${nameValue}"`,
      });

      onRefresh();
      pop();
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Create Failed",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm
            title="Create Search"
            icon={Icon.Plus}
            onSubmit={handleSubmit}
          />
          <Action
            title="Cancel"
            icon={Icon.XMarkCircle}
            onAction={() => pop()}
            shortcut={{ modifiers: [], key: "escape" }}
          />
        </ActionPanel>
      }
    >
      <Form.TextField
        id="name"
        title="Name"
        placeholder="Enter search name"
        value={nameValue}
        onChange={setNameValue}
      />
      <Form.TextField
        id="expression"
        title="Boolean Expression"
        placeholder="tag1 AND tag2 OR tag3"
        value={expressionValue}
        onChange={handleExpressionChange}
        error={expressionError || undefined}
        info={tags && tags.length > 0 ? `Available tags: ${tags.slice(0, 5).join(", ")}${tags.length > 5 ? "..." : ""}` : undefined}
      />
      <Form.TextField
        id="textQuery"
        title="Text Query (Optional)"
        placeholder="Additional text filter"
        value={textQueryValue}
        onChange={setTextQueryValue}
        info="Optional text search to further filter results"
      />
    </Form>
  );
}

interface EditSavedSearchFormProps {
  search: SavedSearch;
  onRefresh: () => void;
}

function EditSavedSearchForm({ search, onRefresh }: EditSavedSearchFormProps) {
  const { pop } = useNavigation();
  const [nameValue, setNameValue] = useState(search.name);
  const [expressionValue, setExpressionValue] = useState(formatBooleanExpression(search.expression));
  const [textQueryValue, setTextQueryValue] = useState(search.text_query || "");
  const [expressionError, setExpressionError] = useState<string>("");
  
  const { data: tags } = useTags();


  const validateExpression = (input: string) => {
    if (!input.trim()) {
      setExpressionError("");
      return;
    }
    
    try {
      parseBooleanExpression(input);
      setExpressionError("");
    } catch (error) {
      setExpressionError("Invalid boolean expression");
    }
  };

  const handleExpressionChange = (value: string) => {
    setExpressionValue(value);
    validateExpression(value);
  };

  const handleSubmit = async () => {
    if (!nameValue.trim() || !expressionValue.trim()) {
      showToast({
        style: Toast.Style.Failure,
        title: "Validation Error",
        message: "Name and expression are required",
      });
      return;
    }

    if (expressionError) {
      showToast({
        style: Toast.Style.Failure,
        title: "Expression Error",
        message: expressionError,
      });
      return;
    }

    try {
      // Delete old search
      await pocketPromptAPI.deleteSavedSearch(search.name);

      // Create new search with updated values
      const expression = parseBooleanExpression(expressionValue);

      await pocketPromptAPI.createSavedSearch({
        name: nameValue.trim(),
        expression,
        textQuery: textQueryValue.trim() || undefined,
      });

      showToast({
        style: Toast.Style.Success,
        title: "Search Updated",
        message: `Updated "${nameValue}"`,
      });

      onRefresh();
      pop();
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Update Failed",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm
            title="Update Search"
            icon={Icon.Checkmark}
            onSubmit={handleSubmit}
          />
          <Action
            title="Cancel"
            icon={Icon.XMarkCircle}
            onAction={() => pop()}
            shortcut={{ modifiers: [], key: "escape" }}
          />
        </ActionPanel>
      }
    >
      <Form.TextField
        id="name"
        title="Name"
        placeholder="Enter search name"
        value={nameValue}
        onChange={setNameValue}
      />
      <Form.TextField
        id="expression"
        title="Boolean Expression"
        placeholder="tag1 AND tag2 OR tag3"
        value={expressionValue}
        onChange={handleExpressionChange}
        error={expressionError || undefined}
        info={tags && tags.length > 0 ? `Available tags: ${tags.slice(0, 5).join(", ")}${tags.length > 5 ? "..." : ""}` : undefined}
      />
      <Form.TextField
        id="textQuery"
        title="Text Query (Optional)"
        placeholder="Additional text filter"
        value={textQueryValue}
        onChange={setTextQueryValue}
        info="Optional text search to further filter results"
      />
    </Form>
  );
}