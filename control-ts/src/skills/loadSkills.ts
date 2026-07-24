import { readdirSync, readFileSync, statSync } from "fs";
import { resolve, join } from "path";
import { parse } from "yaml";
import { projectRoot } from "../utils/projectRoot.js";

const SKILLS_DIR = resolve(projectRoot, "..", ".aether", "skills");

export interface Skill {
  name: string;
  type: "colony" | "domain";
  description: string;
  agent_roles: string[];
  task_keywords: string[];
  domains: string[];
  workflow_triggers: string[];
  priority: string;
  source: string;
  version: string;
  content: string;
}

function findSkillFiles(dir: string): string[] {
  const results: string[] = [];
  let entries: string[];
  try {
    entries = readdirSync(dir);
  } catch {
    return results;
  }
  for (const entry of entries) {
    const fullPath = join(dir, entry);
    try {
      const st = statSync(fullPath);
      if (st.isDirectory()) {
        results.push(...findSkillFiles(fullPath));
      } else if (st.isFile() && entry === "SKILL.md") {
        results.push(fullPath);
      }
    } catch {
      // skip unreadable entries
    }
  }
  return results;
}

function extractFrontmatter(raw: string): { frontmatter: string; body: string } | null {
  if (!raw.startsWith("---")) {
    return null;
  }
  const end = raw.indexOf("---", 3);
  if (end === -1) {
    return null;
  }
  const frontmatter = raw.slice(3, end).trim();
  const body = raw.slice(end + 3).trim();
  return { frontmatter, body };
}

/**
 * Recursively load all SKILL.md files from .aether/skills/colony/ and .aether/skills/domain/.
 * Parses YAML frontmatter and returns an array of Skill objects.
 * Files without frontmatter or with invalid frontmatter are skipped with a warning.
 */
export function loadSkills(): Skill[] {
  const skillFiles = findSkillFiles(SKILLS_DIR);
  const skills: Skill[] = [];
  for (const filePath of skillFiles) {
    let raw: string;
    try {
      raw = readFileSync(filePath, "utf8");
    } catch {
      console.warn(`Skipping unreadable skill file: ${filePath}`);
      continue;
    }
    const extracted = extractFrontmatter(raw);
    if (!extracted) {
      console.warn(`Skipping skill file without frontmatter: ${filePath}`);
      continue;
    }
    let meta: unknown;
    try {
      meta = parse(extracted.frontmatter);
    } catch (err) {
      console.warn(
        `Skipping skill file with invalid frontmatter: ${filePath} — ${err instanceof Error ? err.message : String(err)}`
      );
      continue;
    }
    if (typeof meta !== "object" || meta === null) {
      console.warn(`Skipping skill file with non-object frontmatter: ${filePath}`);
      continue;
    }
    const m = meta as Record<string, unknown>;
    const skill: Skill = {
      name: String(m.name ?? ""),
      type: (m.type === "colony" || m.type === "domain" ? m.type : "colony") as "colony" | "domain",
      description: String(m.description ?? ""),
      agent_roles: Array.isArray(m.agent_roles) ? m.agent_roles.map(String) : [],
      task_keywords: Array.isArray(m.task_keywords) ? m.task_keywords.map(String) : [],
      domains: Array.isArray(m.domains) ? m.domains.map(String) : [],
      workflow_triggers: Array.isArray(m.workflow_triggers) ? m.workflow_triggers.map(String) : [],
      priority: String(m.priority ?? "normal"),
      source: String(m.source ?? ""),
      version: String(m.version ?? ""),
      content: extracted.body,
    };
    skills.push(skill);
  }
  return skills;
}

export interface PheromoneSignal {
  type: string;
  content: string;
}

function scoreSkill(skill: Skill, role: string, task: string, pheromones: PheromoneSignal[]): number {
  let score = 0;
  if (skill.agent_roles.includes(role)) {
    score += 2;
  }
  const taskLower = task.toLowerCase();
  for (const kw of skill.task_keywords) {
    if (taskLower.includes(kw.toLowerCase())) {
      score += 1;
      break;
    }
  }
  for (const p of pheromones) {
    const pLower = p.content.toLowerCase();
    for (const kw of skill.task_keywords) {
      if (pLower.includes(kw.toLowerCase())) {
        score += 1;
        break;
      }
    }
  }
  return score;
}

/**
 * Match skills to a worker by role, task, and active pheromone signals.
 * Returns up to 6 skills (3 colony + 3 domain) sorted by relevance score descending.
 */
export function matchSkillsForWorker(
  role: string,
  task: string,
  pheromones: PheromoneSignal[]
): Skill[] {
  const allSkills = loadSkills();
  const scored = allSkills.map((skill) => ({
    skill,
    score: scoreSkill(skill, role, task, pheromones),
  }));
  scored.sort((a, b) => b.score - a.score);

  const colony = scored.filter((s) => s.skill.type === "colony").slice(0, 3).map((s) => s.skill);
  const domain = scored.filter((s) => s.skill.type === "domain").slice(0, 3).map((s) => s.skill);

  const combined = [...colony, ...domain];
  // If fewer than 6 total, return all; otherwise return exactly 6
  return combined;
}
