# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## tg-dev-agent — Claude Code Skill for Telegram Bot Testing

A Claude Code skill that uses `tgdev` MCP tools to test Telegram bots. Defines workflows for invoking TL methods, monitoring updates, and managing groups.

### Structure

- `SKILL.md` — Skill definition and full workflow (794 lines)
- `evals/evals.json` — Evaluation suite
- `scripts/setup_test_group.sh` — Helper for group setup
- `references/` — Quick-reference docs: common TL methods, error codes, group ops, interactions, monitoring

### Key Constraint

Group creation, member management, and admin promotion require a **user account** session (not bot).
