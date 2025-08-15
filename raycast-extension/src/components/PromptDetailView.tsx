// @ts-nocheck
import {
  ActionPanel,
  Action,
  Detail,
  showToast,
  Toast,
  Clipboard,
  Icon,
  Color,
  confirmAlert,
  Alert,
  useNavigation,
} from "@raycast/api";
import { useState, useEffect } from "react";
import { PocketPrompt } from "../types";
import { pocketPromptAPI } from "../utils/api";
import EditPromptForm from "./EditPromptForm";

interface PromptDetailViewProps {
  prompt: PocketPrompt;
  onRefresh?: () => void;
}

export default function PromptDetailView({
  prompt,
  onRefresh,
}: PromptDetailViewProps) {
  const { push } = useNavigation();
  const [fullPrompt, setFullPrompt] = useState<PocketPrompt>(prompt);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const fetchFullPrompt = async () => {
      // ALWAYS fetch full content for detail view
      // This ensures edit action is never blocked by API calls
      setIsLoading(true);
      try {
        const fullPromptData = await pocketPromptAPI.getPrompt(prompt.ID);
        setFullPrompt(fullPromptData);
      } catch (error) {
        console.error("Failed to fetch full prompt:", error);
        // Keep using the original prompt if fetch fails
        setFullPrompt(prompt);
      } finally {
        setIsLoading(false);
      }
    };

    fetchFullPrompt();
  }, [prompt.ID]);
  const copyPromptToClipboard = async () => {
    try {
      await Clipboard.copy(fullPrompt.Content);
      showToast({
        style: Toast.Style.Success,
        title: "Copied to Clipboard",
        message: fullPrompt.Name,
      });
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Failed to Copy",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const copyRawContent = async () => {
    await Clipboard.copy(fullPrompt.Content);
    showToast({
      style: Toast.Style.Success,
      title: "Copied Raw Content",
      message: fullPrompt.Name,
    });
  };

  const copyAsJSON = async () => {
    const jsonData = JSON.stringify(fullPrompt, null, 2);
    await Clipboard.copy(jsonData);
    showToast({
      style: Toast.Style.Success,
      title: "Copied as JSON",
      message: fullPrompt.Name,
    });
  };

  const copyFilePath = async () => {
    if (fullPrompt.FilePath) {
      await Clipboard.copy(fullPrompt.FilePath);
      showToast({
        style: Toast.Style.Success,
        title: "File Path Copied",
        message: "File path copied to clipboard",
      });
    } else {
      showToast({
        style: Toast.Style.Failure,
        title: "No File Path",
        message: "This prompt doesn't have an associated file path",
      });
    }
  };

  const handleEditPrompt = () => {
    // Edge case: if user clicks edit before content loads, wait briefly
    if (isLoading) {
      showToast({
        style: Toast.Style.Animated,
        title: "Loading...",
        message: "Preparing prompt for editing",
      });
      return;
    }

    // Content is guaranteed to be loaded by useEffect
    // Edit form opens instantly with no API delays
    push(
      <EditPromptForm
        prompt={fullPrompt}
        onSave={(updatedPrompt) => {
          setFullPrompt(updatedPrompt);
          onRefresh?.();
        }}
      />,
    );
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getPromptIcon = () => {
    if (fullPrompt.TemplateRef) {
      return Icon.Document;
    }
    return Icon.Text;
  };

  const buildMarkdownContent = () => {
    // Show only the content in the main area
    if (fullPrompt.Content && fullPrompt.Content.trim()) {
      return fullPrompt.Content;
    } else {
      return `*Loading content...*`;
    }
  };

  const renderWrappedSummary = (summary: string) => {
    // Break long summaries into multiple lines for better display
    const maxLength = 80;
    if (summary.length <= maxLength) {
      return (
        <Detail.Metadata.Label
          title="Summary"
          text={summary}
          icon={Icon.Info}
        />
      );
    }

    // Split into multiple labels if too long
    const words = summary.split(" ");
    const lines: string[] = [];
    let currentLine = "";

    words.forEach((word) => {
      if ((currentLine + " " + word).length <= maxLength) {
        currentLine = currentLine ? currentLine + " " + word : word;
      } else {
        if (currentLine) lines.push(currentLine);
        currentLine = word;
      }
    });
    if (currentLine) lines.push(currentLine);

    return (
      <>
        <Detail.Metadata.Label
          title="Summary"
          text={lines[0]}
          icon={Icon.Info}
        />
        {lines.slice(1).map((line, index) => (
          <Detail.Metadata.Label key={index} title="" text={line} />
        ))}
      </>
    );
  };

  return (
    <Detail
      isLoading={isLoading}
      markdown={buildMarkdownContent()}
      navigationTitle={fullPrompt.Name}
      metadata={
        <Detail.Metadata>
          <Detail.Metadata.Label
            title="Name"
            text={fullPrompt.Name}
            icon={getPromptIcon()}
          />

          {fullPrompt.Summary && renderWrappedSummary(fullPrompt.Summary)}

          <Detail.Metadata.Separator />

          <Detail.Metadata.Label
            title="ID"
            text={fullPrompt.ID}
            icon={Icon.Hashtag}
          />
          <Detail.Metadata.Label
            title="Version"
            text={fullPrompt.Version}
            icon={Icon.Number00}
          />

          {fullPrompt.Tags && fullPrompt.Tags.length > 0 && (
            <Detail.Metadata.TagList title="Tags">
              {fullPrompt.Tags.map((tag) => (
                <Detail.Metadata.TagList.Item
                  key={tag}
                  text={tag}
                  color={Color.Blue}
                />
              ))}
            </Detail.Metadata.TagList>
          )}

          <Detail.Metadata.Separator />

          <Detail.Metadata.Label
            title="Created"
            text={formatDate(fullPrompt.CreatedAt)}
            icon={Icon.Calendar}
          />
          <Detail.Metadata.Label
            title="Updated"
            text={formatDate(fullPrompt.UpdatedAt)}
            icon={Icon.Clock}
          />

          {fullPrompt.FilePath && (
            <>
              <Detail.Metadata.Separator />
              <Detail.Metadata.Label
                title="File Path"
                text={fullPrompt.FilePath}
                icon={Icon.Folder}
              />
            </>
          )}

          {fullPrompt.TemplateRef && (
            <Detail.Metadata.Label
              title="Template"
              text={fullPrompt.TemplateRef}
              icon={Icon.Document}
            />
          )}
        </Detail.Metadata>
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Primary Actions">
            <Action
              title="Edit Prompt"
              icon={Icon.Pencil}
              onAction={handleEditPrompt}
              shortcut={{ modifiers: ["cmd"], key: "e" }}
            />
            <Action
              title="Copy to Clipboard"
              icon={Icon.Clipboard}
              onAction={copyPromptToClipboard}
            />
            <Action
              title="Copy Raw Content"
              icon={Icon.Document}
              onAction={copyRawContent}
              shortcut={{ modifiers: ["cmd"], key: "c" }}
            />
          </ActionPanel.Section>

          <ActionPanel.Section title="Export Actions">
            <Action
              title="Copy as JSON"
              icon={Icon.Code}
              onAction={copyAsJSON}
              shortcut={{ modifiers: ["cmd", "shift"], key: "c" }}
            />
            {fullPrompt.FilePath && (
              <Action
                title="Copy File Path"
                icon={Icon.Folder}
                onAction={copyFilePath}
                shortcut={{ modifiers: ["cmd", "shift"], key: "f" }}
              />
            )}
          </ActionPanel.Section>

          <ActionPanel.Section title="Navigation">
            <Action
              title="Refresh"
              icon={Icon.RotateClockwise}
              onAction={() => onRefresh?.()}
              shortcut={{ modifiers: ["cmd"], key: "r" }}
            />
          </ActionPanel.Section>
        </ActionPanel>
      }
    />
  );
}
