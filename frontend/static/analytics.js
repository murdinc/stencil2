// Enhanced Analytics Tracker with Real-time & E-commerce Support
(function() {
    'use strict';

    // Configuration
    var SESSION_KEY = '_a_sid';
    var VISITOR_KEY = '_a_vid';
    var SESSION_DURATION = 30 * 60 * 1000; // 30 minutes
    var VISITOR_DURATION = 365 * 24 * 60 * 60 * 1000; // 1 year
    var HEARTBEAT_INTERVAL = 30 * 1000; // 30 seconds
    var heartbeatTimer = null;
    var isVisible = true;

    // Visitor ID management (persistent, 1 year)
    function getVisitorId() {
        var stored = localStorage.getItem(VISITOR_KEY);
        var now = Date.now();

        if (stored) {
            try {
                var data = JSON.parse(stored);
                if (now - data.created < VISITOR_DURATION) {
                    return data.id;
                }
            } catch (e) {}
        }

        // Create new visitor ID
        var newId = generateId();
        localStorage.setItem(VISITOR_KEY, JSON.stringify({
            id: newId,
            created: now
        }));
        return newId;
    }

    // Session management (30 minutes)
    function getSessionId() {
        var stored = localStorage.getItem(SESSION_KEY);
        var now = Date.now();

        if (stored) {
            try {
                var data = JSON.parse(stored);
                if (now - data.lastActivity < SESSION_DURATION) {
                    // Update last activity
                    data.lastActivity = now;
                    localStorage.setItem(SESSION_KEY, JSON.stringify(data));
                    return data.id;
                }
            } catch (e) {}
        }

        // Create new session
        var newId = generateId();
        localStorage.setItem(SESSION_KEY, JSON.stringify({
            id: newId,
            lastActivity: now
        }));
        return newId;
    }

    function generateId() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            var r = Math.random() * 16 | 0;
            var v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }

    // Device detection based on screen width
    function getDeviceType() {
        var width = screen.width;
        if (width < 768) return 'mobile';
        if (width < 1024) return 'tablet';
        return 'desktop';
    }

    // Track function
    function track(type, eventName, eventData) {
        var payload = {
            v: getVisitorId(),
            s: getSessionId(),
            t: type || 'p', // 'p' for pageview, 'e' for event, 'h' for heartbeat
            p: location.pathname,
            r: document.referrer || '',
            sw: screen.width,
            sh: screen.height,
            dt: getDeviceType()
        };

        if (type === 'e' && eventName) {
            payload.e = eventName;
            if (eventData) {
                payload.d = eventData;
            }
        }

        // Use sendBeacon for reliability
        if (navigator.sendBeacon) {
            navigator.sendBeacon('/api/v1/track', JSON.stringify(payload));
        } else {
            fetch('/api/v1/track', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
                keepalive: true
            }).catch(function() {});
        }
    }

    // Heartbeat to track active users
    function sendHeartbeat() {
        if (isVisible) {
            track('h'); // Heartbeat type
        }
    }

    function startHeartbeat() {
        if (heartbeatTimer) return;
        heartbeatTimer = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL);
    }

    function stopHeartbeat() {
        if (heartbeatTimer) {
            clearInterval(heartbeatTimer);
            heartbeatTimer = null;
        }
    }

    // Visibility tracking for accurate session duration
    function handleVisibilityChange() {
        if (document.hidden) {
            isVisible = false;
            stopHeartbeat();
        } else {
            isVisible = true;
            startHeartbeat();
            sendHeartbeat(); // Send immediately when tab becomes visible
        }
    }

    // Track initial pageview
    function init() {
        track('p');
        startHeartbeat();
    }

    if (document.readyState === 'complete') {
        init();
    } else {
        window.addEventListener('load', init);
    }

    // Listen for visibility changes
    document.addEventListener('visibilitychange', handleVisibilityChange);

    // Track before page unload
    window.addEventListener('beforeunload', function() {
        stopHeartbeat();
    });

    // Expose global analytics API
    window.analytics = {
        // Track custom events
        track: function(eventName, eventData) {
            track('e', eventName, eventData);
        },

        // E-commerce helpers
        trackAddToCart: function(productId, productName, price, quantity) {
            track('e', 'add_to_cart', {
                product_id: productId,
                product_name: productName,
                price: price,
                quantity: quantity || 1
            });
        },

        trackRemoveFromCart: function(productId) {
            track('e', 'remove_from_cart', {
                product_id: productId
            });
        },

        trackCheckoutStarted: function(cartValue, itemCount) {
            track('e', 'checkout_started', {
                cart_value: cartValue,
                item_count: itemCount
            });
        },

        trackPurchase: function(orderId, total, itemCount) {
            track('e', 'purchase', {
                order_id: orderId,
                total: total,
                item_count: itemCount
            });
        },

        // Content engagement helpers
        trackScrollDepth: function(percentage) {
            track('e', 'scroll_depth', {
                percentage: percentage
            });
        },

        trackClick: function(element, label) {
            track('e', 'click', {
                element: element,
                label: label
            });
        }
    };
})();
