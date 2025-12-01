#!/usr/bin/env zsh
set -euo pipefail

project_root=${0:a:h}/..
cd "$project_root"

echo "Building and installing hugo-revise..."
go install ./cmd/hugo-revise

echo "Preparing demo content..."
rm -rf examples/content/posts/demo* examples/content/posts/.hugo-revise .hugo-revise 2>/dev/null || true
mkdir -p examples/content/posts
cat > examples/content/posts/demo.md <<'EOF'
---
title: "Demo Post"
date: 2024-06-15
---

Hello from demo post.
EOF

echo "Running revise..."
hugo-revise examples/content/posts/demo.md

echo "Tree after revision:"
find examples/content -print | sed 's/^/  /'

echo "Undoing..."
hugo-revise undo

echo "Tree after undo:"
find examples/content -print | sed 's/^/  /'
echo "Done."