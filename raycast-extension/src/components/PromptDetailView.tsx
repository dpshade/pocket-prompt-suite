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
  Form,
  useNavigation,
} from "@raycast/api";
import { useState, useEffect } from "react";
import { PocketPrompt, RenderParams } from "../types";
import { pocketPromptAPI } from "../utils/api";

function VariableForm({
  prompt,
  onSubmit,
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

interface PromptDetailViewProps {
  prompt: PocketPrompt;
  onRefresh?: () => void;
}

export default function PromptDetailView({
  prompt,
  onRefresh,
}: PromptDetailViewProps) {
  const [fullPrompt, setFullPrompt] = useState<PocketPrompt>(prompt);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const fetchFullPrompt = async () => {
      // Only fetch if the current prompt doesn't have content
      if (!prompt.Content || prompt.Content.trim() === "") {
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
      }
    };

    fetchFullPrompt();
  }, [prompt.ID, prompt.Content]);
  const copyPromptToClipboard = async () => {
    try {
      if (fullPrompt.Variables && fullPrompt.Variables.length > 0) {
        return;
      }

      const rendered = await pocketPromptAPI.renderPrompt(fullPrompt.ID);
      await Clipboard.copy(rendered);
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

  const renderWithVariables = async (variables: RenderParams) => {
    try {
      const rendered = await pocketPromptAPI.renderPrompt(
        fullPrompt.ID,
        variables,
      );
      await Clipboard.copy(rendered);
      showToast({
        style: Toast.Style.Success,
        title: "Rendered and Copied",
        message: fullPrompt.Name,
      });
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Failed to Render",
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
    if (fullPrompt.Variables && fullPrompt.Variables.length > 0) {
      return Icon.Gear;
    }
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

          {fullPrompt.Variables && fullPrompt.Variables.length > 0 && (
            <>
              <Detail.Metadata.Separator />
              <Detail.Metadata.Label
                title="Variables"
                text={`${fullPrompt.Variables.length} variables defined`}
                icon={Icon.Gear}
              />
              {fullPrompt.Variables.map((variable) => (
                <Detail.Metadata.Label
                  key={variable.name}
                  title={variable.name}
                  text={`${variable.type}${variable.required ? " (required)" : ""}`}
                  icon={Icon.Text}
                />
              ))}
            </>
          )}

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
            {fullPrompt.Variables && fullPrompt.Variables.length > 0 ? (
              <Action.Push
                title="Fill Variables & Copy"
                icon={Icon.Gear}
                target={
                  <VariableForm
                    prompt={fullPrompt}
                    onSubmit={(variables) => renderWithVariables(variables)}
                  />
                }
              />
            ) : (
              <Action
                title="Copy to Clipboard"
                icon={Icon.Clipboard}
                onAction={copyPromptToClipboard}
              />
            )}
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
