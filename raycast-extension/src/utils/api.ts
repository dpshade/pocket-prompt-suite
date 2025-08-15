import { getPreferenceValues } from "@raycast/api";
import { PocketPrompt, PocketPromptTemplate, ServerStatus, SavedSearch, BooleanExpression } from "../types";

interface Preferences {
  serverUrl: string;
}

function getServerUrl(): string {
  const preferences = getPreferenceValues<Preferences>();
  return preferences.serverUrl || "http://localhost:8080";
}

export class PocketPromptAPI {
  private async request<T>(
    endpoint: string,
    options?: RequestInit,
  ): Promise<T> {
    const baseUrl = getServerUrl();
    const response = await fetch(`${baseUrl}${endpoint}`, {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      ...options,
    });

    if (!response.ok) {
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }

    return response.json() as Promise<T>;
  }

  private async requestText(endpoint: string): Promise<string> {
    const baseUrl = getServerUrl();
    const response = await fetch(`${baseUrl}${endpoint}`, {
      method: "GET",
      headers: {
        Accept: "text/plain",
      },
    });

    if (!response.ok) {
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }

    return response.text();
  }

  async checkHealth(): Promise<ServerStatus> {
    return this.request<ServerStatus>("/health");
  }

  async searchPrompts(query: string): Promise<PocketPrompt[]> {
    const encodedQuery = encodeURIComponent(query);
    return this.request<PocketPrompt[]>(
      `/search?q=${encodedQuery}`,
    );
  }

  async listAllPrompts(): Promise<PocketPrompt[]> {
    return this.request<PocketPrompt[]>("/prompts");
  }

  async listPromptsByPack(packName: string): Promise<PocketPrompt[]> {
    const encodedPack = encodeURIComponent(packName);
    return this.request<PocketPrompt[]>(`/prompts?pack=${encodedPack}`);
  }

  async getPrompt(id: string): Promise<PocketPrompt> {
    return this.request<PocketPrompt>(`/prompts/${id}`);
  }

  async getTags(): Promise<string[]> {
    const tagsText = await this.requestText("/tags");
    return tagsText.split("\n").filter((tag) => tag.trim() !== "");
  }

  async getPromptsByTag(tag: string): Promise<PocketPrompt[]> {
    return this.request<PocketPrompt[]>(
      `/tags/${encodeURIComponent(tag)}`,
    );
  }

  async listTemplates(): Promise<PocketPromptTemplate[]> {
    return this.request<PocketPromptTemplate[]>("/templates");
  }

  async getTemplate(id: string): Promise<PocketPromptTemplate> {
    return this.request<PocketPromptTemplate>(`/templates/${id}`);
  }

  async booleanSearch(expression: string): Promise<PocketPrompt[]> {
    const encodedExpr = encodeURIComponent(expression);
    return this.request<PocketPrompt[]>(
      `/boolean?expr=${encodedExpr}`,
    );
  }

  async hybridSearch(
    fuzzyQuery: string,
    booleanExpr: string,
  ): Promise<PocketPrompt[]> {
    const params = new URLSearchParams();

    if (fuzzyQuery.trim()) {
      params.append("q", fuzzyQuery);
    }
    if (booleanExpr.trim()) {
      params.append("expr", booleanExpr);
    }

    return this.request<PocketPrompt[]>(
      `/search?${params.toString()}`,
    );
  }

  async listSavedSearches(): Promise<string[]> {
    const savedSearchesText = await this.requestText(
      "/saved-searches/list",
    );
    // Parse the "name: expression" format
    return savedSearchesText
      .split("\n")
      .filter((line) => line.trim() !== "")
      .map((line) => line.split(":")[0].trim());
  }

  async listSavedSearchesDetailed(): Promise<SavedSearch[]> {
    return this.request<SavedSearch[]>("/saved-searches/list?format=json");
  }

  async createSavedSearch(savedSearch: {
    name: string;
    expression: BooleanExpression;
    textQuery?: string;
  }): Promise<{ success: boolean; message: string }> {
    return this.request<{ success: boolean; message: string }>(
      "/saved-searches/list",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({
          Name: savedSearch.name,
          Expression: savedSearch.expression,
          TextQuery: savedSearch.textQuery || "",
        }),
      },
    );
  }

  async deleteSavedSearch(name: string): Promise<{ success: boolean; message: string }> {
    const encodedName = encodeURIComponent(name);
    return this.request<{ success: boolean; message: string }>(
      `/saved-searches/delete/${encodedName}`,
      {
        method: "DELETE",
        headers: {
          Accept: "application/json",
        },
      },
    );
  }

  async executeSavedSearch(searchName: string): Promise<PocketPrompt[]> {
    const encodedName = encodeURIComponent(searchName);
    return this.request<PocketPrompt[]>(
      `/saved-search/${encodedName}`,
    );
  }

  async updatePrompt(
    id: string,
    prompt: Partial<PocketPrompt>,
  ): Promise<{ success: boolean; message: string; id: string }> {
    return this.request<{ success: boolean; message: string; id: string }>(
      `/prompts/${id}`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(prompt),
      },
    );
  }

  async getAvailablePacks(): Promise<{ [displayName: string]: string }> {
    const response = await this.request<{ packs: { [displayName: string]: string }; success: boolean }>("/packs?format=json");
    return response.packs;
  }

  async createPrompt(prompt: {
    name: string;
    summary: string;
    content: string;
    tags: string[];
    pack?: string;
  }): Promise<{ success: boolean; message: string; id: string }> {
    // Generate a unique ID based on the name
    const id = prompt.name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "") // Remove special characters
      .replace(/\s+/g, "-") // Replace spaces with hyphens
      .replace(/-+/g, "-") // Collapse multiple hyphens
      .replace(/^-|-$/g, ""); // Remove leading/trailing hyphens

    // Transform to match Go struct field names
    const goPrompt = {
      id: id,
      name: prompt.name,
      summary: prompt.summary,
      content: prompt.content,
      tags: prompt.tags,
      pack: prompt.pack || "personal", // Default to personal library
      version: "1.0.0", // Default version for new prompts
    };

    return this.request<{ success: boolean; message: string; id: string }>(
      "/prompts",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(goPrompt),
      },
    );
  }
}

export const pocketPromptAPI = new PocketPromptAPI();
