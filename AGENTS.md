- Run go tests with "make test"
- Don't add inline comments
- Be concise
- Only use comments when necessary.
- Strive to make the code self-explanatory.
- When comments are used, they should add useful information that is not apparent from the code itself.
- Only lint .go files
- Don't resize images in Draw() or Update() methods, do image resizing when creating a new screen

- After making changes run golangci-lint run to fix any formatting errors.

For Python:
- Use type hints everywhere possible
