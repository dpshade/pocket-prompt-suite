import { getPreferenceValues } from "@raycast/api";
import { PocketPrompt, PocketPromptTemplate, ServerStatus, SavedSearch, BooleanExpression } from "../types";

interface Preferences {
  serverUrl: string;
}

// APIResponse structure from the new API server
interface APIResponse<T> {
  success: boolean;
  data?: T;
  message?: string;
  error?: any;
  timestamp: string;
}

function getServerUrl(): string {
  const preferences = getPreferenceValues<Preferences>();
  return preferences.serverUrl || "http://localhost:8080";
}

function getApiBaseUrl(): string {
  return `${getServerUrl()}/api/v1`;
}

export class PocketPromptAPI {
  private async request<T>(
    endpoint: string,
    options?: RequestInit,
  ): Promise<T> {
    const baseUrl = getApiBaseUrl();
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

    const apiResponse = await response.json() as APIResponse<T>;
    
    if (!apiResponse.success) {
      throw new Error(
        apiResponse.error?.message || apiResponse.message || "API request failed"
      );
    }

    return apiResponse.data as T;
  }


  async checkHealth(): Promise<ServerStatus> {
    return this.request<ServerStatus>("/health");
  }

  async searchPrompts(query: string, packs?: string[]): Promise<PocketPrompt[]> {
    const params = new URLSearchParams();
    params.append("q", query);
    
    if (packs && packs.length > 0) {
      params.append("packs", packs.join(","));
    }
    
    return this.request<PocketPrompt[]>(
      `/search?${params.toString()}`,
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
    return this.request<string[]>("/tags");
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

  async booleanSearch(expression: string, packs?: string[]): Promise<PocketPrompt[]> {
    const params = new URLSearchParams();
    params.append("expr", expression);
    
    if (packs && packs.length > 0) {
      params.append("packs", packs.join(","));
    }
    
    return this.request<PocketPrompt[]>(
      `/boolean-search?${params.toString()}`,
    );
  }

  async hybridSearch(
    fuzzyQuery: string,
    booleanExpr: string,
    packs?: string[],
  ): Promise<PocketPrompt[]> {
    const params = new URLSearchParams();

    if (fuzzyQuery.trim()) {
      params.append("q", fuzzyQuery);
    }
    if (booleanExpr.trim()) {
      params.append("expr", booleanExpr);
    }
    if (packs && packs.length > 0) {
      params.append("packs", packs.join(","));
    }

    return this.request<PocketPrompt[]>(
      `/search?${params.toString()}`,
    );
  }

  async listSavedSearches(): Promise<string[]> {
    const savedSearches = await this.request<string[]>("/saved-searches");
    return savedSearches;
  }

  async listSavedSearchesDetailed(): Promise<SavedSearch[]> {
    return this.request<SavedSearch[]>("/saved-searches?format=json");
  }

  async createSavedSearch(savedSearch: {
    name: string;
    expression: BooleanExpression;
    textQuery?: string;
  }): Promise<{ success: boolean; message: string }> {
    return this.request<{ success: boolean; message: string }>(
      "/saved-searches",
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
      `/saved-searches/${encodedName}`,
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
    const packs = await this.request<{ [displayName: string]: string }>("/packs?format=json");
    return packs;
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
