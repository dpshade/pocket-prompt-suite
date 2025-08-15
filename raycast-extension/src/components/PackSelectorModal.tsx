// @ts-nocheck
import { Action, ActionPanel, Form, Icon, useNavigation } from "@raycast/api";
import { useState } from "react";
import { useCachedPromise } from "@raycast/utils";
import { pocketPromptAPI } from "../utils/api";

interface PackSelectorModalProps {
  selectedPacks: string[];
  onSave: (packs: string[]) => void;
}

export default function PackSelectorModal({ selectedPacks, onSave }: PackSelectorModalProps) {
  const { pop } = useNavigation();
  const isAll = (selectedPacks || []).includes("__all__");
  const [allEnabled, setAllEnabled] = useState<boolean>(isAll);
  const [packs, setPacks] = useState<string[]>(isAll ? [] : selectedPacks || ["personal"]);

  const { data: availablePacks, isLoading } = useCachedPromise(
    async () => pocketPromptAPI.getAvailablePacks(),
    [],
    { initialData: { "Personal Library (default)": "personal" } },
  );

  const items = Object.entries(availablePacks || {}).map(([displayName, packName]) => ({
    id: packName,
    name: displayName,
  }));

  return (
    <Form
      isLoading={isLoading}
      navigationTitle="Select Packs"
      actions={
        <ActionPanel>
          <Action
            title="Save Selection"
            icon={Icon.Check}
            onAction={() => {
              if (allEnabled) {
                onSave(["__all__"]);
              } else {
                onSave(packs.length > 0 ? packs : ["personal"]);
              }
              pop();
            }}
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
      <Form.Checkbox
        id="all"
        label="All Prompts"
        value={allEnabled}
        onChange={setAllEnabled}
        info="Include prompts from all packs"
      />
      <Form.TagPicker
        id="packs"
        title="Packs"
        value={packs}
        onChange={setPacks}
        info="Choose one or more packs to include in search context"
        disabled={allEnabled}
      >
        {items.map((item) => (
          <Form.TagPicker.Item key={item.id} value={item.id} title={item.name} />
        ))}
      </Form.TagPicker>
    </Form>
  );
}


