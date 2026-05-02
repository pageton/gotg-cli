#!/usr/bin/env python3
"""
Package a skill directory into a .skill file (tar.gz archive).

Usage:
    python -m scripts.package_skill <skill_directory>

Creates: <skill_directory>.skill (tar.gz archive)
"""
import os
import sys
import tarfile

def package_skill(skill_dir):
    """Package a skill directory into a .skill file (tar.gz)."""
    if not os.path.isdir(skill_dir):
        print(f"Error: {skill_dir} is not a directory")
        sys.exit(1)

    skill_name = os.path.basename(skill_dir.rstrip('/'))
    output_file = f"{skill_name}.skill"

    # Check for SKILL.md
    skill_md = os.path.join(skill_dir, "SKILL.md")
    if not os.path.exists(skill_md):
        print(f"Warning: {skill_md} not found")

    # Create tar.gz archive
    with tarfile.open(output_file, "w:gz") as tar:
        tar.add(skill_dir, arcname=skill_name)

    abs_path = os.path.abspath(output_file)
    print(f"Created: {abs_path}")
    return abs_path

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python -m scripts.package_skill <skill_directory>")
        sys.exit(1)

    skill_dir = sys.argv[1]
    package_skill(skill_dir)
