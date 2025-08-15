// @ts-nocheck
import {
  ActionPanel,
  Action,
  Form,
  showToast,
  Toast,
  Icon,
  useNavigation,
} from "@raycast/api";
import { useState } from "react";
import { useForm, FormValidation, useCachedPromise } from "@raycast/utils";
import { pocketPromptAPI } from "./utils/api";
import { useTags } from "./hooks/usePocketPrompt";

interface FormValues {
  name: string;
  summary: string;
  content: string;
  tags: string;
  pack: string;
}

interface AddPromptProps {
  onRefresh?: () => void;
}

export default function AddPrompt({ onRefresh }: AddPromptProps) {
  const [isLoading, setIsLoading] = useState(false);
  const { pop } = useNavigation();
  const { data: existingTags } = useTags();
  const { data: availablePacks } = useCachedPromise(
    async () => pocketPromptAPI.getAvailablePacks(),
    [],
    { initialData: { "Personal Library (default)": "personal" } }
  );

  const { handleSubmit, itemProps } = useForm<FormValues>({
    async onSubmit(values) {
      setIsLoading(true);
      try {
        const tags = values.tags
          .split(",")
          .map((tag) => tag.trim())
          .filter((tag) => tag.length > 0);

        const result = await pocketPromptAPI.createPrompt({
          name: values.name,
          summary: values.summary,
          content: values.content,
          tags,
          pack: values.pack,
        });

        if (result.success) {
          showToast({
            style: Toast.Style.Success,
            title: "Prompt Created",
            message: `"${values.name}" has been added to your library`,
          });
          onRefresh?.(); // Refresh the search results
          pop();
        } else {
          showToast({
            style: Toast.Style.Failure,
            title: "Failed to Create Prompt",
            message: result.message || "Unknown error occurred",
          });
        }
      } catch (error) {
        showToast({
          style: Toast.Style.Failure,
          title: "Error Creating Prompt",
          message: error instanceof Error ? error.message : "Unknown error",
        });
      } finally {
        setIsLoading(false);
      }
    },
    initialValues: {
      pack: "personal", // Default to personal library
    },
    validation: {
      name: FormValidation.Required,
      content: FormValidation.Required,
      pack: FormValidation.Required,
    },
  });

  const getTagsPlaceholder = () => {
    if (existingTags && existingTags.length > 0) {
      const exampleTags = existingTags.slice(0, 3).join(", ");
      return `e.g., ${exampleTags}`;
    }
    return "e.g., ai, productivity, coding";
  };

  return (
    <Form
      isLoading={isLoading}
      actions={
        <ActionPanel>
          <Action.SubmitForm
            title="Create Prompt"
            icon={Icon.Plus}
            onSubmit={handleSubmit}
          />
          <Action
            title="Cancel"
            icon={Icon.XMarkCircle}
            onAction={() => pop()}
          />
        </ActionPanel>
      }
    >
      <Form.TextField
        title="Name"
        placeholder="Give your prompt a descriptive name"
        {...itemProps.name}
      />

      <Form.TextField
        title="Summary"
        placeholder="Brief description of what this prompt does (optional)"
        {...itemProps.summary}
      />

      <Form.Dropdown
        title="Pack"
        info="Select which pack to save this prompt to"
        {...itemProps.pack}
      >
        {availablePacks && Object.entries(availablePacks).map(([displayName, packName]) => (
          <Form.Dropdown.Item key={packName} value={packName} title={displayName} />
        ))}
      </Form.Dropdown>

      <Form.TextArea
        title="Content"
        placeholder="Enter your prompt content here..."
        enableMarkdown={true}
        {...itemProps.content}
      />

      <Form.TextField
        title="Tags"
        placeholder={getTagsPlaceholder()}
        info="Comma-separated tags to help organize and find this prompt"
        {...itemProps.tags}
      />

      <Form.Separator />

      <Form.Description text="Your new prompt will be saved to the selected pack and immediately available for search." />
    </Form>
  );
}
