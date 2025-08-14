import { getPreferenceValues } from "@raycast/api";
import { PocketPrompt, PocketPromptTemplate, SearchResult, ServerStatus, RenderParams } from "../types";

interface Preferences {
  serverUrl: string;
}

function getServerUrl(): string {
  const preferences = getPreferenceValues<Preferences>();
  return preferences.serverUrl || "http://localhost:8080";
}

export class PocketPromptAPI {
  private async request<T>(endpoint: string): Promise<T> {
    const baseUrl = getServerUrl();
    const response = await fetch(`${baseUrl}${endpoint}`, {
      method: "GET",
      headers: {
        "Accept": "application/json",
      },
    });

    if (!response.ok) {
      throw new Error(`API request failed: ${response.status} ${response.statusText}`);
    }

    return response.json() as Promise<T>;
  }

  private async requestText(endpoint: string): Promise<string> {
    const baseUrl = getServerUrl();
    const response = await fetch(`${baseUrl}${endpoint}`, {
      method: "GET",
      headers: {
        "Accept": "text/plain",
      },
    });

    if (!response.ok) {
      throw new Error(`API request failed: ${response.status} ${response.statusText}`);
    }

    return response.text();
  }

  async checkHealth(): Promise<ServerStatus> {
    return this.request<ServerStatus>("/health");
  }

  async searchPrompts(query: string): Promise<PocketPrompt[]> {
    const encodedQuery = encodeURIComponent(query);
    return this.request<PocketPrompt[]>(`/pocket-prompt/search?q=${encodedQuery}&format=json`);
  }

  async listAllPrompts(): Promise<PocketPrompt[]> {
    return this.request<PocketPrompt[]>("/pocket-prompt/list?format=json");
  }

  async getPrompt(id: string): Promise<PocketPrompt> {
    return this.request<PocketPrompt>(`/pocket-prompt/get/${id}?format=json`);
  }

  async renderPrompt(id: string, variables?: RenderParams): Promise<string> {
    let endpoint = `/pocket-prompt/render/${id}?format=text`;
    
    if (variables) {
      const params = new URLSearchParams();
      Object.entries(variables).forEach(([key, value]) => {
        params.append(key, String(value));
      });
      endpoint += `&${params.toString()}`;
    }

    return this.requestText(endpoint);
  }

  async getTags(): Promise<string[]> {
    const tagsText = await this.requestText("/pocket-prompt/tags");
    return tagsText.split("\n").filter(tag => tag.trim() !== "");
  }

  async getPromptsByTag(tag: string): Promise<PocketPrompt[]> {
    return this.request<PocketPrompt[]>(`/pocket-prompt/tag/${encodeURIComponent(tag)}?format=json`);
  }

  async listTemplates(): Promise<PocketPromptTemplate[]> {
    return this.request<PocketPromptTemplate[]>("/pocket-prompt/templates?format=json");
  }

  async getTemplate(id: string): Promise<PocketPromptTemplate> {
    return this.request<PocketPromptTemplate>(`/pocket-prompt/template/${id}?format=json`);
  }

  async booleanSearch(expression: string): Promise<PocketPrompt[]> {
    const encodedExpr = encodeURIComponent(expression);
    return this.request<PocketPrompt[]>(`/pocket-prompt/boolean?expr=${encodedExpr}&format=json`);
  }

  async listSavedSearches(): Promise<string[]> {
    const savedSearchesText = await this.requestText("/pocket-prompt/saved-searches/list");
    // Parse the "name: expression" format
    return savedSearchesText.split("\n")
      .filter(line => line.trim() !== "")
      .map(line => line.split(":")[0].trim());
  }

  async executeSavedSearch(searchName: string): Promise<PocketPrompt[]> {
    const encodedName = encodeURIComponent(searchName);
    return this.request<PocketPrompt[]>(`/pocket-prompt/saved-search/${encodedName}?format=json`);
  }
}

export const pocketPromptAPI = new PocketPromptAPI();