// @ts-nocheck
import { Action, ActionPanel, Form, Icon, useNavigation, showToast, Toast } from "@raycast/api";
import { useState, useEffect } from "react";
import { useCachedPromise } from "@raycast/utils";
import { pocketPromptAPI } from "../utils/api";

interface PackSelectorModalProps {
  selectedPacks: string[];
  onSave: (packs: string[]) => void;
}

export default function PackSelectorModal({ selectedPacks, onSave }: PackSelectorModalProps) {
  const { pop } = useNavigation();
  
  // Initialize packs with current selection, filtering out the legacy "__all__" value
  const initialPacks = selectedPacks.filter(p => p !== "__all__");
  const [packs, setPacks] = useState<string[]>(
    initialPacks.length > 0 ? initialPacks : ["personal"]
  );

  const { data: availablePacks, isLoading } = useCachedPromise(
    async () => pocketPromptAPI.getAvailablePacks(),
    [],
    { initialData: { "Personal Library": "personal" } },
  );

  // Create sorted items list with Personal Library first
  const items = Object.entries(availablePacks || {})
    .sort(([aName], [bName]) => {
      // Personal Library always comes first
      if (aName.includes("Personal")) return -1;
      if (bName.includes("Personal")) return 1;
      return aName.localeCompare(bName);
    })
    .map(([displayName, packName]) => ({
      id: packName,
      name: displayName,
    }));

  // Helper to select/deselect all packs
  const toggleAllPacks = () => {
    if (packs.length === items.length) {
      // All selected, deselect all except personal
      setPacks(["personal"]);
    } else {
      // Not all selected, select all
      setPacks(items.map(item => item.id));
    }
  };

  return (
    <Form
      isLoading={isLoading}
      navigationTitle="Select Prompt Packs"
      actions={
        <ActionPanel>
          <Action
            title="Save Selection"
            icon={Icon.Check}
            onAction={() => {
              if (packs.length === 0) {
                showToast({
                  style: Toast.Style.Failure,
                  title: "No Packs Selected",
                  message: "Please select at least one pack",
                });
                return;
              }
              onSave(packs);
              pop();
            }}
          />
          <Action
            title={packs.length === items.length ? "Deselect All" : "Select All"}
            icon={Icon.CheckCircle}
            onAction={toggleAllPacks}
            shortcut={{ modifiers: ["cmd", "shift"], key: "a" }}
          />
          <Action
            title="Cancel"
            icon={Icon.XMarkCircle}
            onAction={() => pop()}
          />
        </ActionPanel>
      }
    >
      <Form.Description 
        title="Active Packs"
        text={`Currently showing prompts from: ${
          packs.length === items.length 
            ? "All packs" 
            : packs.length === 1 && packs[0] === "personal"
            ? "Personal Library only"
            : `${packs.length} pack${packs.length !== 1 ? 's' : ''}`
        }`}
      />
      
      <Form.TagPicker
        id="packs"
        title="Select Packs"
        value={packs}
        onChange={setPacks}
        info="Choose which prompt packs to include in your searches"
        placeholder="Select one or more packs..."
      >
        {items.map((item) => (
          <Form.TagPicker.Item 
            key={item.id} 
            value={item.id} 
            title={item.name}
            icon={item.id === "personal" ? Icon.Person : Icon.Box}
          />
        ))}
      </Form.TagPicker>
      
      <Form.Description 
        text="Tip: Use ⌘⇧A to quickly select or deselect all packs"
      />
    </Form>
  );
}