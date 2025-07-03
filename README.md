# Claude Code Project Template

- All project control in CLAUDE.md and /plan directory
- Uses a universal log and favors simple LLM accessible CLI tools (credit to lucumr.pocoo.org/)
- Assumes playwright-mcp and gemini-cli available. No other MCP tools (credit to simonwillison.net)
- Leverages Makefile for project-wide commands
- Leans heavily on plan.md and task.json. Will bootstrap plan.md if not specified

#### Usage
```
# clone template → strip history → start fresh
git clone --depth 1 https://github.com/org/template.git NEW_PROJECT
cd NEW_PROJECT
rm -rf .git                         # remove template’s history
git init
git add .
git commit -m "init: scaffold from template"
git remote add origin git@github.com:org/NEW_PROJECT.git
git push -u origin main
```

#### Roadmap:
- [Hooks](https://docs.anthropic.com/en/docs/claude-code/hooks)
- [Devcontainer](https://docs.anthropic.com/en/docs/claude-code/devcontainer)