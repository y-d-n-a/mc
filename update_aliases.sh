#!/bin/bash

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Path to .zshrc
ZSHRC="$HOME/.zshrc"

# Remove old aliases if they exist
sed -i.bak '/# mcoder-go aliases start/,/# mcoder-go aliases end/d' "$ZSHRC"

# Add new aliases
cat >> "$ZSHRC" << EOF

# mcoder-go aliases start
export MCODER_DIR="$SCRIPT_DIR"
alias mc="$SCRIPT_DIR/bin/mc"
alias ask="$SCRIPT_DIR/bin/ask"
alias askc="$SCRIPT_DIR/bin/askc"
alias gmfp="$SCRIPT_DIR/bin/mc get 1"
alias mcoder="$SCRIPT_DIR/bin/mc"
# mcoder-go aliases end
EOF

echo "Aliases added to $ZSHRC"
echo "Run 'source ~/.zshrc' or open a new terminal to use the aliases"
echo ""
echo "Available commands:"
echo "  mc get <count> <file|glob> [file|glob ...] [-r] [-- instructions]"
echo "  mc write <index|list> - Write response to disk or list responses"
echo "  mc open [index] - View response(s)"
echo "  mc checkpoint - Set checkpoint at current version"
echo "  mc rollback [n|checkpoint] - Rollback to version"
echo "  mc clear [-y] - Clear workspace"
echo "  mc undo - Undo last write"
echo "  mc ignore <pattern> - Add ignore pattern"
echo "  mc rmignore <pattern> - Remove ignore pattern"
echo "  mc lsignores - List ignore patterns"
echo "  mc model [add|remove] - Manage models"
echo "  mc repeat [count] - Repeat last get call"
echo "  mc prompt <add|delete|update|switch|list> [name] - Manage system prompts"
echo "  mc cost [clear] - Show or clear project costs"
echo "  ask <prompt> - Single query"
echo "  askc - Interactive chat"
