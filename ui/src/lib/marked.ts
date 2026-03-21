declare global {
  interface Window {
    marked?: {
      parse(input: string): string;
    };
  }
}

export function renderMarkdown(input: string): string {
  if (!input) {
    return "";
  }
  return window.marked ? window.marked.parse(input) : input;
}

export {};
