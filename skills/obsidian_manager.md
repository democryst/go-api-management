# Obsidian Knowledge Manager: Vault Sync Protocols

This document defines how we interact with the local Obsidian vault to query and write architectural standards.

## 📂 Vault Reference Path
* **Path**: `/Volumes/SSD990PRO2TB/obsidian-vault`

---

## 🔍 Lookup Workflow (Read Phase)
Before starting any significant architectural change or debugging complex bugs:
1. **Search the Index**: Access `/Volumes/SSD990PRO2TB/obsidian-vault/000_AI_Context_Index.md` to map high-level pillars.
2. **Scan Governance and Tech Specs**: Check the folders `04_Governance/Specs/` and `04_Governance/ADRs/` for structural rules, approved patterns, and historical constraints.
3. **Analyze Engineering Logs**: Review project-specific logs under `02_Projects/<PROJECT>/Engineering-Log.md` for context on current implementations or past setbacks.

---

## 🔄 Synchronize Learnings (Write Phase)
Whenever a unique technical solution, structural fix, or significant architecture decision is made:

### 1. Identify Note Category
* **Project-Specific Engineering Progress** → `02_Projects/<PROJECT>/Engineering-Log.md`
* **Core Technical Skill / Gotcha** → `03_Knowledge/Skills/<SKILL_NAME>.md`
* **Broad Concept / Protocol** → `03_Knowledge/Concepts/<CONCEPT_NAME>.md`
* **Architectural Decisions (ADR)** → `04_Governance/ADRs/adr-NNN.md`

### 2. Note Structure Standards
Every note must contain standard YAML frontmatter:
```markdown
---
title: Note Title
tags: [project/go-api-management, type/skill, status/stable]
type: skill
links: ["[[000_AI_Context_Index]]"]
last_optimized: 2026-05-23
---
```

### 3. Cross-Linking (Wikilinks)
Maximize vault discovery by linking related notes using standard Obsidian wikilinks: `[[Note-Name]]` or `[[Folder/Path/Note-Name|Custom Display Label]]`.

---
*Last Updated: 2026-05-23*
