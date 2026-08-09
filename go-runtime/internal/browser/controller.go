// Package browser implements a browser controller for OVAV agents.
//
// Two modes:
//   - Headless: launches a new Chromium instance (for server-side automation)
//   - Remote: connects to user's existing Chrome via CDP (for live frontend editing)
//
// Uses chromedp (Chrome DevTools Protocol) to provide browser automation tools
// consumable via MCP JSON-RPC 2.0 protocol over stdio.
//
// Tools:
//   - browser_navigate(url) → page HTML
//   - browser_screenshot(selector?) → base64 PNG
//   - browser_click(selector) → success
//   - browser_type(selector, text) → success
//   - browser_get_html(selector?) → HTML string
//   - browser_get_styles(selector) → CSS computed styles
//   - browser_evaluate(js) → JS eval result
//   - browser_inject_css(selector, css) → modify styles live
//   - browser_set_html(selector, html) → modify DOM live
//   - browser_set_attribute(selector, attr, value) → modify attributes
//   - browser_insert_adjacent(selector, position, html) → insert HTML
package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Controller manages a single browser instance.
type Controller struct {
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	allocCtx    context.Context
	allocCancel context.CancelFunc
	active      bool
	headless    bool
	remote      bool
	port        int
	wsEndpoint  string
}

// New creates a browser controller.
func New(opts ...Option) *Controller {
	c := &Controller{
		headless: true,
		port:     9222,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Option configures the browser controller.
type Option func(*Controller)

// WithHeadless sets headless mode (default true).
func WithHeadless(h bool) Option {
	return func(c *Controller) { c.headless = h }
}

// WithPort sets the CDP port (default 9222).
func WithPort(p int) Option {
	return func(c *Controller) { c.port = p }
}

// Start launches a new headless browser.
func (c *Controller) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active {
		return fmt.Errorf("browser: already running")
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", c.headless),
		chromedp.WindowSize(1920, 1080),
	)

	c.allocCtx, c.allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	c.ctx, c.cancel = chromedp.NewContext(c.allocCtx)

	if err := chromedp.Run(c.ctx); err != nil {
		c.allocCancel()
		return fmt.Errorf("browser: start failed: %w", err)
	}

	c.active = true
	return nil
}

// ConnectRemote connects to an existing Chrome instance via CDP.
// Chrome must be running with: chrome --remote-debugging-port=9222
// This uses the USER's real Chrome with their cookies, sessions, and extensions.
// Connects to the FIRST EXISTING TAB to avoid creating new windows.
func (c *Controller) ConnectRemote(wsEndpoint string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active {
		return nil // Already connected, reuse
	}

	if wsEndpoint == "" {
		wsEndpoint = fmt.Sprintf("http://127.0.0.1:%d", c.port)
	}

	// Strategy 1: Connect to first existing tab (avoids creating new windows)
	wsURL, err := c.getFirstTabWS(wsEndpoint)
	if err != nil {
		// Strategy 2: Fall back to browser-level connection
		wsURL, err = c.getWebSocketURL(wsEndpoint)
		if err != nil {
			return fmt.Errorf("browser: cannot connect to Chrome at %s: %w\n"+
				"  → Start Chrome with: chrome --remote-debugging-port=%d",
				wsEndpoint, err, c.port)
		}
	}

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	ctx, cancel := chromedp.NewContext(allocCtx)

	// Pass a no-op action so chromedp.Run completes the init handshake.
	// Empty chromedp.Run blocks indefinitely because the first target is about:blank.
	if err := chromedp.Run(ctx, chromedp.Sleep(100*time.Millisecond)); err != nil {
		allocCancel()
		return fmt.Errorf("browser: remote connect failed: %w", err)
	}

	c.allocCtx = allocCtx
	c.allocCancel = allocCancel
	c.ctx = ctx
	c.cancel = cancel
	c.wsEndpoint = wsEndpoint
	c.remote = true
	c.active = true

	// If we connected to about:blank, navigate to the app
	// This prevents the "about:blank tab" accumulation issue
	go c.navigateToAppIfBlank("http://localhost:5173")

	return nil
}

// navigateToAppIfBlank navigates to the app URL if current page is about:blank.
func (c *Controller) navigateToAppIfBlank(appURL string) {
	c.mu.Lock()
	if !c.active {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	var currentURL string
	if err := chromedp.Run(c.ctx, chromedp.Evaluate("window.location.href", &currentURL)); err != nil {
		return
	}
	if currentURL == "about:blank" || currentURL == "" {
		chromedp.Run(c.ctx, chromedp.Navigate(appURL))
	}
}

// getFirstTabWS returns the WebSocket URL of the first existing tab.
// This avoids creating new browser contexts (new windows).
func (c *Controller) getFirstTabWS(endpoint string) (string, error) {
	resp, err := http.Get(endpoint + "/json/list")
	if err != nil {
		return "", fmt.Errorf("cannot list tabs at %s", endpoint)
	}
	defer resp.Body.Close()

	var tabs []struct {
		ID                   string `json:"id"`
		URL                  string `json:"url"`
		Title                string `json:"title"`
		WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
		Type                 string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return "", fmt.Errorf("cannot parse tabs")
	}

	// Find first "page" type tab with REAL content (skip about:blank)
	for _, tab := range tabs {
		if tab.Type == "page" && tab.WebSocketDebuggerUrl != "" &&
			tab.URL != "about:blank" && tab.URL != "" {
			return tab.WebSocketDebuggerUrl, nil
		}
	}

	// Fallback: find any page tab (even about:blank) but clean up others
	for _, tab := range tabs {
		if tab.Type == "page" && tab.WebSocketDebuggerUrl != "" {
			// Clean up OTHER about:blank tabs in background
			go c.cleanupBlankTabs(endpoint, tab.ID)
			return tab.WebSocketDebuggerUrl, nil
		}
	}

	return "", fmt.Errorf("no page tabs found")
}

// cleanupBlankTabs closes about:blank tabs except the one we're using.
func (c *Controller) cleanupBlankTabs(endpoint, keepID string) {
	resp, err := http.Get(endpoint + "/json/list")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tabs []struct {
		ID   string `json:"id"`
		URL  string `json:"url"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return
	}

	for _, tab := range tabs {
		if tab.ID != keepID && tab.Type == "page" &&
			(tab.URL == "about:blank" || tab.URL == "") {
			// Close the about:blank tab via CDP
			http.Get(endpoint + "/json/close/" + tab.ID)
		}
	}
}

// getWebSocketURL retrieves the WebSocket debugger URL from Chrome's CDP endpoint.
func (c *Controller) getWebSocketURL(endpoint string) (string, error) {
	// Try /json/version first
	resp, err := http.Get(endpoint + "/json/version")
	if err != nil {
		return "", fmt.Errorf("connection refused at %s", endpoint)
	}
	defer resp.Body.Close()

	var info struct {
		WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("invalid response from Chrome CDP")
	}

	if info.WebSocketDebuggerUrl == "" {
		return "", fmt.Errorf("no WebSocket URL in Chrome CDP response")
	}

	return info.WebSocketDebuggerUrl, nil
}

// ConnectToTab connects to a specific tab by URL pattern or index.
func (c *Controller) ConnectToTab(urlPattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.remote || c.wsEndpoint == "" {
		return fmt.Errorf("browser: must ConnectRemote first")
	}

	// List all tabs
	resp, err := http.Get(c.wsEndpoint + "/json/list")
	if err != nil {
		return fmt.Errorf("browser: cannot list tabs: %w", err)
	}
	defer resp.Body.Close()

	var tabs []struct {
		ID                   string `json:"id"`
		URL                  string `json:"url"`
		Title                string `json:"title"`
		WebsocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return fmt.Errorf("browser: cannot parse tabs")
	}

	// Find matching tab
	for _, tab := range tabs {
		if strings.Contains(tab.URL, urlPattern) || strings.Contains(tab.Title, urlPattern) {
			// Connect to this specific tab
			allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), tab.WebsocketDebuggerUrl)
			ctx, cancel := chromedp.NewContext(allocCtx)

			if err := chromedp.Run(ctx); err != nil {
				allocCancel()
				return fmt.Errorf("browser: tab connect failed: %w", err)
			}

			c.allocCancel()
			c.cancel()
			c.allocCtx = allocCtx
			c.allocCancel = allocCancel
			c.ctx = ctx
			c.cancel = cancel
			return nil
		}
	}

	return fmt.Errorf("browser: no tab matching '%s' found", urlPattern)
}

// Stop disconnects from the browser.
// If remote mode, does NOT close Chrome (user's browser stays open).
func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return nil
	}

	c.cancel()
	if c.allocCancel != nil {
		c.allocCancel()
	}
	c.active = false
	c.remote = false
	return nil
}

// Navigate loads a URL and returns the page HTML.
func (c *Controller) Navigate(url string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	var html string
	err := chromedp.Run(c.ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(1*time.Second),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return "", fmt.Errorf("browser: navigate failed: %w", err)
	}
	return html, nil
}

// Screenshot captures the page or an element as a base64 PNG.
func (c *Controller) Screenshot(selector string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	var buf []byte
	if selector == "" {
		if err := chromedp.Run(c.ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
			return "", fmt.Errorf("browser: screenshot failed: %w", err)
		}
	} else {
		if err := chromedp.Run(c.ctx, chromedp.Screenshot(selector, &buf, chromedp.NodeVisible)); err != nil {
			return "", fmt.Errorf("browser: element screenshot failed: %w", err)
		}
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// Click clicks an element by CSS selector.
func (c *Controller) Click(selector string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	return chromedp.Run(c.ctx,
		chromedp.WaitVisible(selector),
		chromedp.Click(selector),
	)
}

// Type types text into an input element.
func (c *Controller) Type(selector, text string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	return chromedp.Run(c.ctx,
		chromedp.WaitVisible(selector),
		chromedp.Focus(selector),
		chromedp.Clear(selector),
		chromedp.SendKeys(selector, text),
	)
}

// GetHTML returns the HTML of the page or a specific element.
func (c *Controller) GetHTML(selector string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	if selector == "" {
		selector = "html"
	}

	var html string
	err := chromedp.Run(c.ctx,
		chromedp.OuterHTML(selector, &html),
	)
	if err != nil {
		return "", fmt.Errorf("browser: get_html failed: %w", err)
	}
	return html, nil
}

// ComputedStyle represents a CSS computed style key-value pair.
type ComputedStyle struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// GetComputedStyles returns computed CSS styles for an element.
func (c *Controller) GetComputedStyles(selector string) ([]ComputedStyle, error) {
	if !c.active {
		return nil, fmt.Errorf("browser: not started")
	}

	var styles []ComputedStyle
	err := chromedp.Run(c.ctx,
		chromedp.WaitVisible(selector),
		chromedp.Evaluate(fmt.Sprintf(`
			(() => {
				const el = document.querySelector(%q);
				if (!el) return [];
				const cs = getComputedStyle(el);
				const props = Array.from(cs).slice(0, 50);
				return props.map(p => ({property: p, value: cs.getPropertyValue(p)}));
			})()
		`, selector), &styles),
	)
	if err != nil {
		return nil, fmt.Errorf("browser: get_styles failed: %w", err)
	}
	return styles, nil
}

// Evaluate executes JavaScript in the page and returns the result.
func (c *Controller) Evaluate(js string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	var result string
	err := chromedp.Run(c.ctx,
		chromedp.Evaluate(js, &result),
	)
	if err != nil {
		return "", fmt.Errorf("browser: evaluate failed: %w", err)
	}
	return result, nil
}

// === DOM MODIFICATION TOOLS (Live Frontend Editing) ===

// InjectCSS injects CSS styles into an element or the page.
// selector: CSS selector ("body" for page-wide, ".class" for specific)
// css: CSS properties to apply (e.g. "background: red; font-size: 20px")
func (c *Controller) InjectCSS(selector, css string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found: %s';
			el.style.cssText += '; %s';
			return 'ok';
		})()
	`, selector, selector, css)

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// SetInnerHTML replaces the inner HTML of an element.
func (c *Controller) SetInnerHTML(selector, html string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	// Escape the HTML for safe injection
	escaped, _ := json.Marshal(html)
	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.innerHTML = %s;
			return 'ok';
		})()
	`, selector, string(escaped))

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// SetOuterHTML replaces the entire element with new HTML.
func (c *Controller) SetOuterHTML(selector, html string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	escaped, _ := json.Marshal(html)
	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.outerHTML = %s;
			return 'ok';
		})()
	`, selector, string(escaped))

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// SetAttribute sets an attribute on an element.
func (c *Controller) SetAttribute(selector, attr, value string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	escapedVal, _ := json.Marshal(value)
	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.setAttribute(%q, %s);
			return 'ok';
		})()
	`, selector, attr, string(escapedVal))

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// InsertAdjacentHTML inserts HTML at a position relative to an element.
// position: "beforebegin", "afterbegin", "beforeend", "afterend"
func (c *Controller) InsertAdjacentHTML(selector, position, html string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	escaped, _ := json.Marshal(html)
	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.insertAdjacentHTML(%q, %s);
			return 'ok';
		})()
	`, selector, position, string(escaped))

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// RemoveElement removes an element from the DOM.
func (c *Controller) RemoveElement(selector string) error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}

	js := fmt.Sprintf(`
		(() => {
			const el = document.querySelector(%q);
			if (!el) return 'element not found';
			el.remove();
			return 'ok';
		})()
	`, selector)

	var result string
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
}

// Reload reloads the current page.
func (c *Controller) Reload() error {
	if !c.active {
		return fmt.Errorf("browser: not started")
	}
	return chromedp.Run(c.ctx, chromedp.Reload())
}

// GetURL returns the current page URL.
func (c *Controller) GetURL() (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	var url string
	err := chromedp.Run(c.ctx, chromedp.Evaluate("window.location.href", &url))
	return url, err
}

// GetTitle returns the current page title.
func (c *Controller) GetTitle() (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	var title string
	err := chromedp.Run(c.ctx, chromedp.Evaluate("document.title", &title))
	return title, err
}

// ListTabs returns all open tabs in the connected Chrome.
func (c *Controller) ListTabs() ([]Tab, error) {
	if !c.remote || c.wsEndpoint == "" {
		return nil, fmt.Errorf("browser: not in remote mode")
	}

	resp, err := http.Get(c.wsEndpoint + "/json/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tabs []Tab
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return nil, err
	}
	return tabs, nil
}

// Tab represents a Chrome tab.
type Tab struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

// Status returns the current browser state.
func (c *Controller) Status() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := map[string]interface{}{
		"active":   c.active,
		"headless": c.headless,
		"remote":   c.remote,
		"port":     c.port,
	}
	if c.remote {
		status["wsEndpoint"] = c.wsEndpoint
	}
	return status
}

// IsPortOpen checks if a port is listening (for Chrome debugging).
func IsPortOpen(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// TypeReact types text into a React controlled input using nativeInputValueSetter.
// This triggers React's state updates properly (unlike standard Type which uses DOM events).
func (c *Controller) TypeReact(selector, text string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	js := fmt.Sprintf(`(() => {
		var el = document.querySelector(%q);
		if (!el) return 'element not found: %s';
		var setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set;
		setter.call(el, %q);
		el.dispatchEvent(new Event("input", {bubbles: true}));
		el.dispatchEvent(new Event("change", {bubbles: true}));
		return "ok: " + el.value;
	})()`, selector, selector, text)

	var result string
	err := chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", fmt.Errorf("browser: type_react failed: %w", err)
	}
	return result, nil
}

// ClickReact clicks an element using React-compatible event dispatching.
func (c *Controller) ClickReact(selector string) (string, error) {
	if !c.active {
		return "", fmt.Errorf("browser: not started")
	}

	js := fmt.Sprintf(`(() => {
		var el = document.querySelector(%q);
		if (!el) return 'element not found: %s';
		el.dispatchEvent(new MouseEvent("click", {bubbles: true, cancelable: true}));
		return "clicked: " + el.tagName;
	})()`, selector, selector)

	var result string
	err := chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", fmt.Errorf("browser: click_react failed: %w", err)
	}
	return result, nil
}
