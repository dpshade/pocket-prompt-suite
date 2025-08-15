// @ts-nocheck
import {
  ActionPanel,
  Action,
  Form,
  showToast,
  Toast,
  useNavigation,
  Icon,
} from "@raycast/api";
import { useState } from "react";
import { useCachedPromise } from "@raycast/utils";
import { PocketPrompt } from "../types";
import { pocketPromptAPI } from "../utils/api";

interface EditPromptFormProps {
  prompt: PocketPrompt;
  onSave?: (updatedPrompt: PocketPrompt) => void;
}

export default function EditPromptForm({
  prompt,
  onSave,
}: EditPromptFormProps) {
  const { pop } = useNavigation();
  const [isLoading, setIsLoading] = useState(false);

  // Form state - prompt should always have full content now
  const [name, setName] = useState(prompt.Name || "");
  const [summary, setSummary] = useState(prompt.Summary || "");
  const [content, setContent] = useState(prompt.Content || "");
  const [tagsText, setTagsText] = useState(prompt.Tags?.join(", ") || "");
  const [templateRef, setTemplateRef] = useState(prompt.TemplateRef || "");
  const [pack, setPack] = useState(prompt.Pack || "personal");

  // Load available packs
  const { data: availablePacks } = useCachedPromise(
    async () => pocketPromptAPI.getAvailablePacks(),
    [],
    { initialData: { "Personal Library (default)": "personal" } }
  );

  const handleSave = async () => {
    // Validation
    if (!name.trim()) {
      showToast({
        style: Toast.Style.Failure,
        title: "Name Required",
        message: "Prompt name cannot be empty",
      });
      return;
    }

    if (!content.trim()) {
      showToast({
        style: Toast.Style.Failure,
        title: "Content Required",
        message: "Prompt content cannot be empty",
      });
      return;
    }

    setIsLoading(true);

    try {
      // Parse tags from comma-separated string
      const tags = tagsText
        .split(",")
        .map((tag) => tag.trim())
        .filter((tag) => tag.length > 0);

      // Prepare update data
      const updateData: Partial<PocketPrompt> = {
        Name: name.trim(),
        Summary: summary.trim(),
        Content: content.trim(),
        Tags: tags,
        Pack: pack,
      };

      // Only include TemplateRef if it's not empty
      if (templateRef.trim()) {
        updateData.TemplateRef = templateRef.trim();
      }

      // Call API to update prompt
      const result = await pocketPromptAPI.updatePrompt(prompt.ID, updateData);

      if (result.success) {
        showToast({
          style: Toast.Style.Success,
          title: "Prompt Updated",
          message: `${name} has been updated successfully`,
        });

        // Fetch the updated prompt data
        const updatedPrompt = await pocketPromptAPI.getPrompt(prompt.ID);
        onSave?.(updatedPrompt);

        // Navigate back
        pop();
      } else {
        throw new Error("Update failed");
      }
    } catch (error) {
      showToast({
        style: Toast.Style.Failure,
        title: "Update Failed",
        message:
          error instanceof Error ? error.message : "Unknown error occurred",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    pop();
  };

  return (
    <Form
      isLoading={isLoading}
      navigationTitle={`Edit: ${prompt.Name}`}
      actions={
        <ActionPanel>
          <ActionPanel.Section>
            <Action
              title="Save Changes"
              icon={Icon.Check}
              onAction={handleSave}
              shortcut={{ modifiers: ["cmd"], key: "s" }}
            />
            <Action
              title="Cancel"
              icon={Icon.XMarkCircle}
              onAction={handleCancel}
              shortcut={{ modifiers: ["cmd"], key: "." }}
            />
          </ActionPanel.Section>
        </ActionPanel>
      }
    >
      <Form.Description
        text={`Editing prompt: ${prompt.ID} (v${prompt.Version})`}
      />

      <Form.Separator />

      <Form.TextField
        id="name"
        title="Name"
        placeholder="Enter prompt name..."
        value={name}
        onChange={setName}
        info="The display name for this prompt"
      />

      <Form.TextArea
        id="summary"
        title="Summary"
        placeholder="Enter a brief description..."
        value={summary}
        onChange={setSummary}
        info="A brief description of what this prompt does"
      />

      <Form.Dropdown
        id="pack"
        title="Pack"
        value={pack}
        onChange={setPack}
        info="Select which pack to save this prompt to"
      >
        {availablePacks && Object.entries(availablePacks).map(([displayName, packName]) => (
          <Form.Dropdown.Item key={packName} value={packName} title={displayName} />
        ))}
      </Form.Dropdown>

      <Form.TextField
        id="tags"
        title="Tags"
        placeholder="tag1, tag2, tag3..."
        value={tagsText}
        onChange={setTagsText}
        info="Comma-separated list of tags for organization"
      />

      <Form.TextField
        id="templateRef"
        title="Template Reference"
        placeholder="Optional template ID..."
        value={templateRef}
        onChange={setTemplateRef}
        info="Reference to a template (optional)"
      />

      <Form.Separator />

      <Form.TextArea
        id="content"
        title="Prompt Content"
        placeholder="Write your prompt here... Use Markdown for formatting."
        value={content}
        onChange={setContent}
        enableMarkdown={true}
        info="The main prompt content - use Markdown for rich formatting"
      />

      <Form.Separator />

      <Form.Description
        text={`Created: ${new Date(prompt.CreatedAt).toLocaleDateString()} • Last Updated: ${new Date(prompt.UpdatedAt).toLocaleDateString()}`}
      />
    </Form>
  );
}
