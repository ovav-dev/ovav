#!/usr/bin/env python3
"""
Content Extractor — HTML to clean text using stdlib only.

No bs4, no lxml, no external dependencies.
Extracts: title, main text, publish date, author hints.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from html.parser import HTMLParser


@dataclass
class ExtractedContent:
    url: str
    title: str = ""
    text: str = ""
    publish_date: str = ""
    author: str = ""
    word_count: int = 0


class _TextExtractor(HTMLParser):
    """HTML parser that extracts text, title, and metadata."""

    def __init__(self):
        super().__init__()
        self.text_parts: list[str] = []
        self.title: str = ""
        self.in_title: bool = False
        self.in_script: bool = False
        self.in_style: bool = False
        self.skip_tags: set[str] = {"script", "style", "noscript", "svg", "canvas"}
        self.meta_date: str = ""
        self.meta_author: str = ""

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]):
        tag_lower = tag.lower()
        if tag_lower in self.skip_tags:
            if tag_lower == "script":
                self.in_script = True
            elif tag_lower == "style":
                self.in_style = True
        elif tag_lower == "title":
            self.in_title = True

        attrs_dict = {k.lower(): v for k, v in attrs if v}
        if tag_lower == "meta":
            prop = attrs_dict.get("property", "") or attrs_dict.get("name", "")
            content = attrs_dict.get("content", "")
            if "date" in prop.lower() or prop in ("article:published_time", "pubdate"):
                self.meta_date = content
            elif "author" in prop.lower():
                self.meta_author = content

    def handle_endtag(self, tag: str):
        tag_lower = tag.lower()
        if tag_lower == "script":
            self.in_script = False
        elif tag_lower == "style":
            self.in_style = False
        elif tag_lower == "title":
            self.in_title = False

    def handle_data(self, data: str):
        if self.in_script or self.in_style:
            return
        if self.in_title:
            self.title += data.strip()
        else:
            stripped = data.strip()
            if stripped:
                self.text_parts.append(stripped)

    def get_text(self) -> str:
        return "\n".join(self.text_parts)


def extract(html: str, url: str = "") -> ExtractedContent:
    """Extract clean text content from HTML string."""
    parser = _TextExtractor()
    try:
        parser.feed(html)
    except Exception:
        pass

    text = parser.get_text()
    # Normalize whitespace
    text = re.sub(r"\n{3,}", "\n\n", text)
    text = re.sub(r"[ \t]{2,}", " ", text)

    # Try to find publish date in text patterns
    date_patterns = [
        r"\b(20[2-3]\d[-/][01]\d[-/][0-3]\d)\b",
        r"\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{1,2},? 20[2-3]\d\b",
    ]
    found_date = parser.meta_date
    if not found_date:
        for pat in date_patterns:
            m = re.search(pat, text[:2000], re.IGNORECASE)
            if m:
                found_date = m.group(1)
                break

    return ExtractedContent(
        url=url,
        title=parser.title[:300] if parser.title else "",
        text=text,
        publish_date=found_date,
        author=parser.meta_author,
        word_count=len(text.split()),
    )
