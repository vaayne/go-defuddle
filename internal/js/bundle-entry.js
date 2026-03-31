// Bundle entry point for QuickJS — HTML extraction only (no markdown)
// Bundles defuddle + linkedom into a single self-contained JS file
// Markdown conversion is handled on the Go side via html-to-markdown
//
// NOTE: polyfills.js must be loaded before this bundle (Buffer, atob, URL, self)

import { parseHTML } from 'linkedom';
import { Defuddle } from '../../defuddle/src/defuddle';

function parseLinkedomHTML(html, url) {
	const { document } = parseHTML(html);
	const doc = document;
	if (!doc.styleSheets) doc.styleSheets = [];
	if (doc.defaultView && !doc.defaultView.getComputedStyle) {
		doc.defaultView.getComputedStyle = () => ({ display: '' });
	}
	if (url) doc.URL = url;
	return document;
}

/**
 * Parse HTML string and return extracted content as JSON string.
 */
globalThis.defuddleParse = function(html, url, optionsJson) {
	try {
		const options = optionsJson ? JSON.parse(optionsJson) : {};
		delete options.markdown;
		delete options.separateMarkdown;

		const doc = parseLinkedomHTML(html, url || 'about:blank');
		const pageUrl = url || doc.URL || 'about:blank';

		const defuddle = new Defuddle(doc, {
			...options,
			url: pageUrl
		});

		const result = defuddle.parse();
		return JSON.stringify(result);
	} catch (e) {
		return JSON.stringify({
			error: String(e.message || e),
			stack: e.stack || '',
			content: '', title: '', description: '', domain: '',
			favicon: '', image: '', language: '', published: '',
			author: '', site: '', wordCount: 0, parseTime: 0,
		});
	}
};
