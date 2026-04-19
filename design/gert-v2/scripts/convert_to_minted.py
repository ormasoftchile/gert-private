#!/usr/bin/env python3
"""Convert verbatim/lstlisting blocks to minted across all section files."""

import re
import os
import sys
import shutil

SECTIONS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), 'sections')

def classify_content(content):
    """Determine the best minted language for a code block, or None to leave as-is."""
    c = content.strip()

    # --- Leave as verbatim early: ASCII trees, error messages, CLI output ---
    if re.search(r'[├└─│]', c): return None  # ASCII tree → verbatim
    if re.search(r'^[✓✗]', c, re.MULTILINE): return None  # CLI output → verbatim
    if re.search(r'^(Run:|Runbook:|Actor:|Started:|Completed:|RUN_ID\s)', c, re.MULTILINE): return None
    if re.search(r'^(Migration report|WARN:|CHANGE\]|ADD\])', c, re.MULTILINE): return None

    # --- Go ---
    if re.search(r'\btype \w+ (interface|struct)\s*\{', c): return 'go'
    if re.search(r'^func ', c, re.MULTILINE): return 'go'
    if re.search(r'^(package |import \()', c, re.MULTILINE): return 'go'
    # Go API calls (span.X, attribute.X, baggage.X, trace.X)
    if re.search(r'^(span\.|attribute\.|baggage\.|trace\.|otel\.)', c, re.MULTILINE): return 'go'

    # --- JSON (starts with { or [ and has quoted keys) ---
    if re.search(r'^\s*[\{\[]', c) and '"' in c and ':' in c:
        return 'json'
    # JSON fragments: top-level "key": { ... } style (event payloads)
    if re.search(r'^"(data|payload|result|error|params|metadata)"\s*:\s*\{', c, re.MULTILINE): return 'json'
    # JSON-RPC pairs with Request:/Response: prefix → verbatim (mixed)
    if re.search(r'^(Request:|Response:)', c, re.MULTILINE): return None

    # --- YAML (key: value patterns with indentation or list items) ---
    if re.search(r'^(id|name|type|steps|runbook|version|on|action|tool|input|output|schema|governance|config|env|args|cmd|shell|timeout|retry|approve|condition|default|branches|policies|rules|redact|capture|with|from|vars|provider|transport|url|port|host|method|path|headers|auth|tls|cert|key|inputs|outputs):', c, re.MULTILINE):
        return 'yaml'
    if re.search(r'^\w[\w_-]*:\s*\S', c, re.MULTILINE) and re.search(r'^  \w', c, re.MULTILINE):
        return 'yaml'
    if re.search(r'^\s*-\s+\w', c, re.MULTILINE) and ':' in c:
        return 'yaml'
    if re.search(r'^\$schema:', c, re.MULTILINE): return 'yaml'

    # --- Bash / shell commands ---
    if re.search(r'^\$ ', c, re.MULTILINE): return 'bash'  # $ prompt
    if re.search(r'^(export |gert |go |make |git |curl |kubectl |docker |./)', c, re.MULTILINE):
        return 'bash'
    if re.search(r'^[A-Z_]+=\S', c, re.MULTILINE) and ':' not in c: return 'bash'

    # --- OTel attribute style: key.key = "value" // comment ---
    if re.search(r'^\w[\w.]+\s+=\s+', c, re.MULTILINE) and '//' in c: return 'text'
    if re.search(r'^\w[\w.]+\s+=\s+', c, re.MULTILINE) and '|' in c: return 'text'

    # --- Leave as verbatim: HTTP headers, file paths, OTel attrs, error messages ---
    return None  # keep as verbatim

def has_lstlisting_options(begin_line):
    """Extract caption and label from lstlisting options if present."""
    m = re.search(r'\[([^\]]*)\]', begin_line)
    if not m: return None, None
    opts = m.group(1)
    caption_m = re.search(r'caption=\{([^}]*)\}', opts)
    label_m = re.search(r'label=\{([^}]*)\}', opts)
    caption = caption_m.group(1) if caption_m else None
    label = label_m.group(1) if label_m else None
    return caption, label

def convert_file(filepath):
    """Convert a single .tex file. Returns (new_content, n_converted)."""
    with open(filepath, 'r') as f:
        content = f.read()

    n_converted = 0
    # Process verbatim blocks
    def replace_verbatim(m):
        nonlocal n_converted
        block_content = m.group(1)
        lang = classify_content(block_content)
        if lang is None:
            return m.group(0)  # keep as-is
        n_converted += 1
        return f'\\begin{{minted}}{{{lang}}}{block_content}\\end{{minted}}'

    # Process lstlisting blocks (may have options)
    def replace_lstlisting(m):
        nonlocal n_converted
        full = m.group(0)
        opts_str = m.group(1) or ''   # optional [options] group
        block_content = m.group(2)    # code content group
        
        # Check for explicit language
        lang_m = re.search(r'language=(\w+)', opts_str, re.IGNORECASE)
        explicit_lang = lang_m.group(1).lower() if lang_m else None
        
        # Normalize explicit language
        if explicit_lang in ('yaml', 'yml'): explicit_lang = 'yaml'
        elif explicit_lang in ('golang', 'go'): explicit_lang = 'go'
        elif explicit_lang in ('json',): explicit_lang = 'json'
        elif explicit_lang in ('bash', 'sh', 'shell'): explicit_lang = 'bash'
        elif explicit_lang is not None:
            explicit_lang = None  # unknown explicit lang, auto-detect
        
        lang = explicit_lang or classify_content(block_content)
        if lang is None:
            return full  # keep as-is
        
        caption, label = has_lstlisting_options(opts_str) if opts_str else (None, None)
        n_converted += 1
        
        if caption:
            result = (f'\\begin{{listing}}[H]\n'
                      f'\\begin{{minted}}{{{lang}}}{block_content}\\end{{minted}}\n'
                      f'\\caption{{{caption}}}\n')
            if label:
                result += f'\\label{{{label}}}\n'
            result += '\\end{listing}'
            return result
        else:
            return f'\\begin{{minted}}{{{lang}}}{block_content}\\end{{minted}}'

    # Apply verbatim replacements
    new_content = re.sub(
        r'\\begin\{verbatim\}(.*?)\\end\{verbatim\}',
        replace_verbatim,
        content,
        flags=re.DOTALL
    )

    # Apply lstlisting replacements
    new_content = re.sub(
        r'\\begin\{lstlisting\}(\[[^\]]*\])?(.*?)\\end\{lstlisting\}',
        replace_lstlisting,
        new_content,
        flags=re.DOTALL
    )

    return new_content, n_converted

def main():
    files = [f for f in sorted(os.listdir(SECTIONS_DIR))
             if f.endswith('.tex') and not f.endswith('.bak')]

    total_converted = 0
    file_counts = {}

    for fname in files:
        fpath = os.path.join(SECTIONS_DIR, fname)
        new_content, n = convert_file(fpath)
        if n > 0:
            # Backup
            shutil.copy2(fpath, fpath + '.mintbak')
            with open(fpath, 'w') as out:
                out.write(new_content)
            file_counts[fname] = n
            total_converted += n
            print(f'  {fname}: converted {n} blocks')
        else:
            print(f'  {fname}: nothing to convert')

    print(f'\nTotal: {total_converted} blocks converted across {len(file_counts)} files')
    return total_converted

if __name__ == '__main__':
    main()
