const CACHE_NAME = 'squad-v1';
const STATIC_ASSETS = [
    '/',
    '/assets/bundle.js',
    '/assets/bundle.css',
    '/assets/index.css',
    '/assets/reset.css',
    '/assets/github.min.css',
    '/assets/react.production.min.js',
    '/assets/react-dom.production.min.js',
    '/assets/markdown-it.min.js',
    '/assets/highlight.min.js',
    '/assets/site.webmanifest',
    '/assets/favicon-32x32.png',
    '/assets/favicon-16x16.png',
    '/assets/apple-touch-icon.png'
];

// Only these GET routes are cached for offline reading; everything else is network only.
const CACHEABLE_API_ROUTES = [
    /^\/api\/users\/me\/$/,
    /^\/api\/bots\/$/,
    /^\/api\/bots\/\d+\/threads\/$/,
    /^\/api\/threads\/\d+\/$/,
    /^\/api\/bots\/\d+\/files\/$/,
    /^\/api\/bots\/\d+\/files\/[^/]+\/$/,
    /^\/api\/attachments\/\d+\/$/
];

self.addEventListener('install', event => {
    event.waitUntil(handleInstall());
});

self.addEventListener('activate', event => {
    event.waitUntil(handleActivate());
});

self.addEventListener('fetch', event => {
    // Live chat streams and writes go straight to the network, a worker in the middle would only buffer or break them.
    const isEventStream = (event.request.headers.get('Accept') || '').includes('text/event-stream');
    if (event.request.method !== 'GET' || isEventStream) {
        return;
    }

    event.respondWith(handleFetch(event));
});

self.addEventListener('message', event => {
    if (event.data?.type === 'CLEAR_CACHE') {
        event.waitUntil(clearAllCaches());
    }
});

async function handleInstall() {
    const cache = await caches.open(CACHE_NAME);

    // cache.addAll() rejects atomically, so one stale filename would silently disable the entire service worker.
    // Caching per asset lets it degrade instead.
    await Promise.all(
        STATIC_ASSETS.map(async asset => {
            try {
                await cache.add(new Request(asset, { cache: 'reload' }));
            } catch (error) {
                console.error(`[sw] failed to precache ${asset}`, error);
            }
        })
    );

    await self.skipWaiting();
}

async function handleActivate() {
    await clearOldCaches();
    await self.clients.claim();
}

async function handleFetch(event) {
    const request = event.request;
    const path = new URL(request.url).pathname;

    if (path.startsWith('/api/')) {
        return handleApiFetch(event, request, path);
    }

    return handleAssetFetch(event, request);
}

// Network first
async function handleApiFetch(event, request, path) {
    const isCacheable = CACHEABLE_API_ROUTES.some(route => route.test(path));

    if (!isCacheable) {
        return fetch(request);
    }

    try {
        const res = await fetch(request);
        if (res.ok && res.redirected !== true) {
            event.waitUntil(cacheResponse(request, res.clone()));
        }
        return res;
    } catch (error) {
        const cached = await caches.match(request);
        if (cached) {
            return cached;
        }
        return buildOfflineResponse();
    }
}

// no-cache makes the browser check with the server, since assets served with Last-Modified and no Cache-Control can otherwise come back from the HTTP cache unchanged.
async function handleAssetFetch(event, request) {
    try {
        const res = await fetch(request, { cache: 'no-cache' });
        // A redirected response may not be the page that was asked for, so it is never cached under this URL.
        if (res.ok && res.redirected !== true) {
            event.waitUntil(cacheResponse(request, res.clone()));
        }
        return res;
    } catch (error) {
        const cached = await caches.match(request);
        if (cached) {
            return cached;
        }

        if (request.mode === 'navigate' || request.destination === 'document') {
            const index = await caches.match('/') || await caches.match('/assets/index.html');
            if (index) {
                return index;
            }
        }

        return buildOfflineResponse();
    }
}

function buildOfflineResponse() {
    return new Response('Offline', {
        status: 503,
        statusText: 'Service Unavailable',
        headers: { 'Content-Type': 'text/plain' }
    });
}

async function cacheResponse(request, response) {
    try {
        const cache = await caches.open(CACHE_NAME);
        await cache.put(request, response);
    } catch (error) {
        console.error(`[sw] failed to cache ${request.url}`, error);
    }
}

// Unlike clearOldCaches, this drops the current cache too, which logout needs.
async function clearAllCaches() {
    const cacheNames = await caches.keys();
    await Promise.all(
        cacheNames
            .filter(cacheName => cacheName.startsWith('squad-'))
            .map(cacheName => caches.delete(cacheName))
    );
}

async function clearOldCaches() {
    const cacheNames = await caches.keys();
    await Promise.all(
        cacheNames
            .filter(cacheName => cacheName !== CACHE_NAME && cacheName.startsWith('squad-'))
            .map(cacheName => caches.delete(cacheName))
    );
}