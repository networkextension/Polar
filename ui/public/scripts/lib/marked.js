export function renderMarkdown(input) {
    if (!input) {
        return "";
    }
    return window.marked ? window.marked.parse(input) : input;
}
