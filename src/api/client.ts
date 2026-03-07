import pkg from "../../package.json";
import type { Config } from "../config.ts";
import type {
  ResultsResponse,
  StrokeData,
  StrokeDataResponse,
  UserProfile,
  UserResponse,
  Workout,
} from "../models.ts";

export class C2Client {
  private baseURL: string;
  private token: string;

  constructor(baseURL: string, token: string) {
    this.baseURL = baseURL;
    this.token = token;
  }

  static fromConfig(cfg: Config): C2Client {
    return new C2Client(cfg.api.base_url, cfg.api.token);
  }

  private async get(path: string): Promise<unknown> {
    const url = this.baseURL + path;
    const resp = await fetch(url, {
      headers: {
        Authorization: `Bearer ${this.token}`,
        "User-Agent": `c2/${pkg.version}`,
      },
      signal: AbortSignal.timeout(30_000),
    });

    if (!resp.ok) {
      throw new Error(`API error (${resp.status}) from ${path}`);
    }

    return resp.json();
  }

  async getUser(): Promise<UserProfile> {
    const resp = (await this.get("/api/users/me")) as UserResponse;
    return resp.data;
  }

  async getResults(from: string, to: string, page: number): Promise<ResultsResponse> {
    const params = new URLSearchParams({ type: "rower", page: String(page) });
    if (from) params.set("from", from);
    if (to) params.set("to", to);
    return (await this.get(`/api/users/me/results?${params}`)) as ResultsResponse;
  }

  async getAllResults(from: string, to: string): Promise<Workout[]> {
    const all: Workout[] = [];
    let page = 1;

    for (;;) {
      const resp = await this.getResults(from, to, page);
      all.push(...resp.data);

      const hasMore =
        resp.meta?.pagination != null &&
        resp.meta.pagination.current_page < resp.meta.pagination.total_pages;

      if (!hasMore || resp.data.length === 0) break;
      page++;
    }

    return all;
  }

  async getStrokes(workoutId: number): Promise<StrokeData[]> {
    const path = `/api/users/me/results/${workoutId}/strokes`;
    const resp = (await this.get(path)) as StrokeDataResponse;
    return resp.data;
  }
}
