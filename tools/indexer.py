#!/usr/bin/env python3
import os
from datetime import datetime

# Configuration
WORKSPACE_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MANIFEST_PATH = os.path.join(WORKSPACE_ROOT, "project_manifest.md")

IGNORE_DIRS = {".git", "__pycache__", ".idea", ".vscode", "node_modules", "vendor", "bin"}
IGNORE_FILES = {".DS_Store", "project_manifest.md"}

def build_tree(dir_path, prefix=""):
    """Recursively builds an ASCII representation of the directory tree."""
    tree_lines = []
    try:
        entries = sorted(os.scandir(dir_path), key=lambda e: (not e.is_dir(), e.name.lower()))
    except OSError:
        return tree_lines

    # Filter entries
    filtered_entries = []
    for entry in entries:
        if entry.is_dir() and entry.name in IGNORE_DIRS:
            continue
        if entry.is_file() and entry.name in IGNORE_FILES:
            continue
        filtered_entries.append(entry)

    count = len(filtered_entries)
    for i, entry in enumerate(filtered_entries):
        is_last = (i == count - 1)
        connector = "└── " if is_last else "├── "
        
        if entry.is_dir():
            tree_lines.append(f"{prefix}{connector}{entry.name}/")
            new_prefix = prefix + ("    " if is_last else "│   ")
            tree_lines.extend(build_tree(entry.path, new_prefix))
        else:
            tree_lines.append(f"{prefix}{connector}{entry.name}")
            
    return tree_lines

def get_file_list(dir_path):
    """Walks the directory and aggregates metadata for all files."""
    files_data = []
    for root, dirs, files in os.walk(dir_path):
        # Exclude ignored directories in-place to avoid walking into them
        dirs[:] = [d for d in dirs if d not in IGNORE_DIRS]
        
        for file in sorted(files):
            if file in IGNORE_FILES:
                continue
            
            abs_path = os.path.join(root, file)
            rel_path = os.path.relpath(abs_path, WORKSPACE_ROOT)
            
            try:
                stat_info = os.stat(abs_path)
                size_bytes = stat_info.st_size
                mtime = datetime.fromtimestamp(stat_info.st_mtime).strftime("%Y-%m-%d %H:%M:%S")
            except OSError:
                size_bytes = 0
                mtime = "Unknown"

            files_data.append({
                "path": rel_path,
                "name": file,
                "size": size_bytes,
                "modified": mtime
            })
            
    return files_data

def main():
    print(f"Indexing workspace: {WORKSPACE_ROOT}...")
    
    # 1. Build ASCII Tree
    tree_lines = [os.path.basename(WORKSPACE_ROOT) + "/"]
    tree_lines.extend(build_tree(WORKSPACE_ROOT))
    ascii_tree = "\n".join(tree_lines)
    
    # 2. Get flat list with metadata
    files_data = get_file_list(WORKSPACE_ROOT)
    
    # 3. Write manifest
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    with open(MANIFEST_PATH, "w", encoding="utf-8") as f:
        f.write(f"# Project Manifest: go-api-management\n\n")
        f.write(f"Generated at: `{timestamp}`\n\n")
        
        f.write(f"## 🏛️ Directory Tree Structure\n\n")
        f.write(f"```text\n")
        f.write(ascii_tree + "\n")
        f.write(f"```\n\n")
        
        f.write(f"## 📂 Cataloged Repository Files\n\n")
        f.write(f"| File Name | Rel Path | Size (Bytes) | Last Modified | Architectural Purpose |\n")
        f.write(f"| :--- | :--- | :--- | :--- | :--- |\n")
        
        for file in files_data:
            # Infer purpose based on directories/extensions
            purpose = "Repository Core Metadata"
            if file["path"].startswith("skills/"):
                purpose = "Operational Skill Guideline / Blueprint"
            elif file["path"].startswith("tools/"):
                purpose = "Developer Tooling / Script"
            elif file["path"].endswith(".go"):
                purpose = "Go Source Code / Business Logic"
                
            f.write(f"| `{file['name']}` | [`{file['path']}`](file://{WORKSPACE_ROOT}/{file['path']}) | {file['size']} | {file['modified']} | {purpose} |\n")
            
    print(f"Successfully generated {MANIFEST_PATH}")

if __name__ == "__main__":
    main()
