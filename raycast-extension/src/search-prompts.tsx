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
import { useUnifiedSearch, useServerHealth, useTags, useSavedSearches } from "./hooks/usePocketPrompt";
import { PocketPrompt, RenderParams } from "./types";
import { pocketPromptAPI } from "./utils/api";
import { analyzeSearchQuery, formatBooleanExpression } from "./utils/searchDetection";
import PromptDetailView from "./components/PromptDetailView";

function VariableForm({ 
  prompt, 
  onSubmit 
}: { 
  prompt: PocketPrompt; 
  onSubmit: (variables: RenderParams) => void;
}) {
  const { pop } = useNavigation();
  const [variables, setVariables] = useState<RenderParams>({});

  const handleSubmit = () => {
    onSubmit(variables);
    pop();
  };

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm title="Render Prompt" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.Description text={`Fill in variables for: ${prompt.Name}`} />
      {prompt.Variables?.map((variable) => (
        <Form.TextField
          key={variable.name}
          id={variable.name}
          title={variable.name}
          placeholder={variable.default?.toString() || ""}
          info={variable.description}
          value={variables[variable.name]?.toString() || ""}
          onChange={(value) =>
            setVariables((prev) => ({ ...prev, [variable.name]: value }))
          }
        />
      ))}
    </Form>
  );
}

export default function SearchPrompts() {
  const [searchText, setSearchText] = useState("");
  const [selectedFilter, setSelectedFilter] = useState<string>("");
  
  const { data: serverHealth, isLoading: healthLoading, error: healthError } = useServerHealth();
  const { data: tags } = useTags();
  const { data: savedSearches } = useSavedSearches();

  // Smart search analysis
  const searchAnalysis = useMemo(() => {
    if (selectedFilter.startsWith("saved:")) {
      return { type: 'saved' as const, query: searchText, searchName: selectedFilter.replace("saved:", "") };
    }
    if (selectedFilter.startsWith("tag:")) {
      const tag = selectedFilter.replace("tag:", "");
      return { type: 'boolean' as const, query: tag };
    }
    
    const analysis = analyzeSearchQuery(searchText);
    return { 
      type: analysis.type, 
      query: analysis.type === 'boolean' ? formatBooleanExpression(searchText) : searchText 
    };
  }, [searchText, selectedFilter]);

  const { data: prompts, isLoading, error, revalidate } = useUnifiedSearch(
    searchAnalysis.query,
    searchAnalysis.type,
    searchAnalysis.searchName
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
      if (prompt.Variables && prompt.Variables.length > 0) {
        return;
      }
      
      const rendered = await pocketPromptAPI.renderPrompt(prompt.ID);
      await Clipboard.copy(rendered);
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

  const renderWithVariables = async (prompt: PocketPrompt, variables: RenderParams) => {
    try {
      const rendered = await pocketPromptAPI.renderPrompt(prompt.ID, variables);
      await Clipboard.copy(rendered);
      showToast({
        style: Toast.Style.Success,
        title: "Rendered and Copied",
        message: prompt.Name,
      });
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Failed to Render",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };


  const getAccessories = (prompt: PocketPrompt) => {
    const accessories = [];
    
    if (prompt.Variables && prompt.Variables.length > 0) {
      accessories.push({ text: `${prompt.Variables.length} vars`, icon: Icon.Gear });
    }
    
    if (prompt.Tags && prompt.Tags.length > 0) {
      accessories.push({ text: prompt.Tags.slice(0, 2).join(", "), icon: Icon.Tag });
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
    return "Search prompts...";
  };

  const getEmptyViewContent = () => {
    if (selectedFilter.startsWith("saved:")) {
      const searchName = selectedFilter.replace("saved:", "");
      return {
        title: "No Results",
        description: `Saved search "${searchName}" returned no results`,
        icon: Icon.Bookmark
      };
    }
    
    if (selectedFilter.startsWith("tag:")) {
      const tag = selectedFilter.replace("tag:", "");
      return {
        title: "No Prompts",
        description: `No prompts found with tag "${tag}"`,
        icon: Icon.Tag
      };
    }

    if (!searchText.trim()) {
      return {
        title: "Search Your Prompts",
        description: 
          `Start typing to search, or use the filter dropdown.\n\n` +
          `• Fuzzy search: "machine learning"\n` +
          `• Boolean search: "ai AND agent"\n` +
          `• Complex logic: "(design OR ui) AND NOT test"\n\n` +
          `Use Ctrl+P to access saved searches and tags.`,
        icon: Icon.MagnifyingGlass
      };
    }

    if (searchAnalysis.type === 'boolean') {
      return {
        title: "No Boolean Results",
        description: `Boolean expression "${searchAnalysis.query}" returned no results`,
        icon: Icon.Code
      };
    }

    return {
      title: "No Results",
      description: `No prompts match "${searchText}"`,
      icon: Icon.ExclamationMark
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
          placeholder="All Prompts"
          onChange={(value) => setSelectedFilter(value || "")}
        >
          <List.Dropdown.Section title="All">
            <List.Dropdown.Item title="All Prompts" value="" />
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
              ...(searchAnalysis.type === 'boolean' ? [{ text: "Boolean", icon: Icon.Code }] : [])
            ]}
            actions={
              <ActionPanel>
                <ActionPanel.Section title="Prompt Actions">
                  {prompt.Variables && prompt.Variables.length > 0 ? (
                    <Action.Push
                      title="Fill Variables & Copy"
                      icon={Icon.Gear}
                      target={
                        <VariableForm
                          prompt={prompt}
                          onSubmit={(variables) => renderWithVariables(prompt, variables)}
                        />
                      }
                    />
                  ) : (
                    <Action
                      title="Copy to Clipboard"
                      icon={Icon.Clipboard}
                      onAction={() => copyPromptToClipboard(prompt)}
                    />
                  )}
                  <Action.Push
                    title="Show Details"
                    icon={Icon.Eye}
                    target={<PromptDetailView prompt={prompt} onRefresh={revalidate} />}
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
                    title="Force Boolean Search"
                    icon={Icon.Code}
                    onAction={() => {
                      if (searchText.trim()) {
                        setSearchText(formatBooleanExpression(searchText));
                      }
                    }}
                    shortcut={{ modifiers: ["cmd"], key: "b" }}
                  />
                  <Action
                    title="Clear Filters"
                    icon={Icon.Trash}
                    onAction={() => {
                      setSelectedFilter("");
                      setSearchText("");
                    }}
                    shortcut={{ modifiers: ["cmd", "shift"], key: "k" }}
                  />
                </ActionPanel.Section>
                <ActionPanel.Section title="Navigation">
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