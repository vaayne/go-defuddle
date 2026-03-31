// Polyfills for QuickJS (no Node.js built-ins)
// Must be loaded BEFORE the defuddle bundle

if (typeof globalThis.self === 'undefined') {
	globalThis.self = globalThis;
}

// Minimal Buffer polyfill — only base64 decode is needed by htmlparser2/entities
if (typeof globalThis.Buffer === 'undefined') {
	globalThis.Buffer = {
		from(str, encoding) {
			if (encoding === 'base64') {
				const binary = atob(str);
				return {
					toString(enc) {
						return binary;
					}
				};
			}
			return { toString() { return str; } };
		}
	};
}

// QuickJS has no URL constructor — provide a minimal polyfill
if (typeof globalThis.URL === 'undefined') {
	globalThis.URL = class URL {
		constructor(url, base) {
			if (base && !url.match(/^[a-zA-Z][a-zA-Z0-9+.-]*:/)) {
				const baseUrl = typeof base === 'string' ? new URL(base) : base;
				if (url.startsWith('//')) {
					url = baseUrl.protocol + url;
				} else if (url.startsWith('/')) {
					url = baseUrl.origin + url;
				} else {
					const basePath = baseUrl.pathname.replace(/\/[^\/]*$/, '/');
					url = baseUrl.origin + basePath + url;
				}
			}
			const match = url.match(/^([a-zA-Z][a-zA-Z0-9+.-]*):\/\/([^/?#:]*)(:(\d+))?(\/[^?#]*)?(\?[^#]*)?(#.*)?$/);
			if (!match) {
				// Handle scheme-only URLs like about:blank, data:, javascript:
				const schemeOnly = url.match(/^([a-zA-Z][a-zA-Z0-9+.-]*):(.*)$/);
				if (schemeOnly) {
					this.protocol = schemeOnly[1] + ':';
					this.hostname = '';
					this.port = '';
					this.pathname = schemeOnly[2] || '';
					this.search = '';
					this.hash = '';
					this.host = '';
					this.origin = 'null';
					this.href = url;
					return;
				}
				throw new TypeError('Invalid URL: ' + url);
			}
			this.protocol = match[1] + ':';
			this.hostname = match[2] || '';
			this.port = match[4] || '';
			this.pathname = match[5] || '/';
			this.search = match[6] || '';
			this.hash = match[7] || '';
			this.host = this.port ? this.hostname + ':' + this.port : this.hostname;
			this.origin = this.protocol + '//' + this.host;
			this.href = this.origin + this.pathname + this.search + this.hash;
		}
		get searchParams() {
			const params = new Map();
			if (this.search) {
				this.search.slice(1).split('&').forEach(pair => {
					const [k, v] = pair.split('=');
					params.set(decodeURIComponent(k), decodeURIComponent(v || ''));
				});
			}
			return { get(k) { return params.get(k) || null; }, has(k) { return params.has(k); } };
		}
		toString() { return this.href; }
	};
}

// performance.now polyfill for QuickJS
if (typeof globalThis.performance === 'undefined') {
	globalThis.performance = { now() { return Date.now(); } };
}

// QuickJS has no atob — provide one if missing
if (typeof globalThis.atob === 'undefined') {
	globalThis.atob = function(b64) {
		const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
		let result = '';
		let i = 0;
		b64 = b64.replace(/[^A-Za-z0-9+/=]/g, '');
		while (i < b64.length) {
			const a = chars.indexOf(b64.charAt(i++));
			const b = chars.indexOf(b64.charAt(i++));
			const c = chars.indexOf(b64.charAt(i++));
			const d = chars.indexOf(b64.charAt(i++));
			const n = (a << 18) | (b << 12) | (c << 6) | d;
			result += String.fromCharCode((n >> 16) & 0xFF);
			if (c !== 64) result += String.fromCharCode((n >> 8) & 0xFF);
			if (d !== 64) result += String.fromCharCode(n & 0xFF);
		}
		return result;
	};
}
