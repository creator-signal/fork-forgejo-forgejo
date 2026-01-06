import {wrapSelectionWithCharacter} from './ComboMarkdownEditor.js';

function createTextareaWithSelection(value, selectionStart, selectionEnd) {
  const textarea = document.createElement('textarea');
  textarea.value = value;
  document.body.append(textarea);
  textarea.setSelectionRange(selectionStart, selectionEnd);
  return textarea;
}

describe('ComboMarkdownEditor wrap selection', () => {
  afterEach(() => {
    for (const el of document.body.querySelectorAll('textarea')) {
      el.remove();
    }
  });

  test('wraps selected text with backticks', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '`');

    expect(result).toBe(true);
    expect(textarea.value).toBe('`hello` world');
    expect(textarea.selectionStart).toBe(1);
    expect(textarea.selectionEnd).toBe(6);
  });

  test('wraps selected text with asterisks', () => {
    const textarea = createTextareaWithSelection('hello world', 6, 11);
    const result = wrapSelectionWithCharacter(textarea, '*');

    expect(result).toBe(true);
    expect(textarea.value).toBe('hello *world*');
    expect(textarea.selectionStart).toBe(7);
    expect(textarea.selectionEnd).toBe(12);
  });

  test('wraps selected text with underscores', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '_');

    expect(result).toBe(true);
    expect(textarea.value).toBe('_hello_ world');
  });

  test('wraps selected text with tildes for strikethrough', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '~');

    expect(result).toBe(true);
    expect(textarea.value).toBe('~hello~ world');
  });

  test('wraps selected text with square brackets', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '[');

    expect(result).toBe(true);
    expect(textarea.value).toBe('[hello] world');
  });

  test('wraps selected text with curly braces', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '{');

    expect(result).toBe(true);
    expect(textarea.value).toBe('{hello} world');
  });

  test('wraps selected text with parentheses', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '(');

    expect(result).toBe(true);
    expect(textarea.value).toBe('(hello) world');
  });

  test('returns false when no selection', () => {
    const textarea = createTextareaWithSelection('hello world', 5, 5);
    const result = wrapSelectionWithCharacter(textarea, '`');

    expect(result).toBe(false);
    expect(textarea.value).toBe('hello world');
  });

  test('returns false for unsupported characters', () => {
    const textarea = createTextareaWithSelection('hello world', 0, 5);
    const result = wrapSelectionWithCharacter(textarea, '"');

    expect(result).toBe(false);
    expect(textarea.value).toBe('hello world');
  });

  test('wraps text in middle of content', () => {
    const textarea = createTextareaWithSelection('the quick brown fox', 4, 9);
    const result = wrapSelectionWithCharacter(textarea, '`');

    expect(result).toBe(true);
    expect(textarea.value).toBe('the `quick` brown fox');
  });

  test('preserves selection after wrapping for continued editing', () => {
    const textarea = createTextareaWithSelection('bold text here', 0, 4);
    wrapSelectionWithCharacter(textarea, '*');

    expect(textarea.value).toBe('*bold* text here');
    // Selection should be on the wrapped content (excluding wrapper chars)
    expect(textarea.selectionStart).toBe(1);
    expect(textarea.selectionEnd).toBe(5);

    // User can now press * again to make it bold (**)
    wrapSelectionWithCharacter(textarea, '*');
    expect(textarea.value).toBe('**bold** text here');
  });
});
