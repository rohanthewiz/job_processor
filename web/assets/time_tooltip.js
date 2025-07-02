// Time tooltip functionality with event delegation
(function() {
    'use strict';
    
    let tooltip;
    let currentTarget = null;
    
    // Initialize tooltip element
    function initializeTooltip() {
        if (tooltip) return; // Already initialized
        
        // Create tooltip element
        tooltip = document.createElement('div');
        tooltip.className = 'time-tooltip';
        tooltip.style.cssText = `
            position: absolute;
            background: rgba(0, 0, 0, 0.9);
            color: white;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 14px;
            pointer-events: none;
            z-index: 1000;
            display: none;
            white-space: nowrap;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
        `;
        document.body.appendChild(tooltip);
    }
    
    // Convert UTC time string to local time
    function convertToLocalTime(utcTimeStr) {
        // Parse the UTC time string (format: "2006-01-02 15:04 MST" or "2006-01-02 15:04:05 MST")
        const parts = utcTimeStr.match(/(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}(?::\d{2})?)\s+(\w+)/);
        if (!parts) {
            console.error('Failed to parse time string:', utcTimeStr);
            return null;
        }
        
        const [_, dateStr, timeStr, tz] = parts;
        
        // Since Go formats UTC times with "UTC" as the timezone, 
        // we can safely parse it as UTC
        // If timeStr already has seconds, don't add :00
        const isoTimeStr = timeStr.includes(':') && timeStr.split(':').length === 2 
            ? `${timeStr}:00` 
            : timeStr;
        const utcDateTime = new Date(`${dateStr}T${isoTimeStr}Z`);
        
        if (isNaN(utcDateTime.getTime())) {
            console.error('Invalid date:', `${dateStr}T${isoTimeStr}Z`);
            return null;
        }
        
        // Format local time
        const options = {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
            timeZoneName: 'short'
        };
        
        const localTimeStr = utcDateTime.toLocaleString(undefined, options);
        const localTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
        
        return {
            utc: utcTimeStr,
            local: localTimeStr,
            timezone: localTz
        };
    }
    
    // Show tooltip for the given element
    function showTooltipForElement(element) {
        if (!tooltip || !element) return;
        
        const utcTime = element.textContent.trim();
        const localTime = convertToLocalTime(utcTime);
        
        if (!localTime) return;
        
        currentTarget = element;
        
        tooltip.innerHTML = `
            <div style="margin-bottom: 4px;"><strong>UTC:</strong> ${localTime.utc}</div>
            <div><strong>Local:</strong> ${localTime.local}</div>
            <div style="font-size: 12px; opacity: 0.8; margin-top: 4px;">Timezone: ${localTime.timezone}</div>
        `;
        
        const rect = element.getBoundingClientRect();
        const tooltipHeight = 80; // Approximate height
        
        // Position tooltip above the element
        tooltip.style.left = rect.left + 'px';
        tooltip.style.top = (rect.top - tooltipHeight - 5) + 'px';
        tooltip.style.display = 'block';
        
        // Adjust if tooltip goes off screen
        setTimeout(() => {
            const tooltipRect = tooltip.getBoundingClientRect();
            if (tooltipRect.top < 0) {
                // Show below instead
                tooltip.style.top = (rect.bottom + 5) + 'px';
            }
            if (tooltipRect.left + tooltipRect.width > window.innerWidth) {
                tooltip.style.left = (window.innerWidth - tooltipRect.width - 10) + 'px';
            }
        }, 0);
    }
    
    // Hide tooltip
    function hideTooltip() {
        if (tooltip) {
            tooltip.style.display = 'none';
            currentTarget = null;
        }
    }
    
    // Handle mouse events using event delegation
    function handleMouseEnter(e) {
        const timestamp = e.target.closest('.timestamp');
        if (timestamp) {
            showTooltipForElement(timestamp);
        }
    }
    
    function handleMouseLeave(e) {
        const timestamp = e.target.closest('.timestamp');
        if (timestamp && timestamp === currentTarget) {
            hideTooltip();
        }
    }
    
    // Setup event delegation on document body
    function setupEventDelegation() {
        // Use event delegation on document body to catch all timestamp elements
        document.body.addEventListener('mouseenter', handleMouseEnter, true);
        document.body.addEventListener('mouseleave', handleMouseLeave, true);
        
        // Style all timestamp elements
        const style = document.createElement('style');
        style.textContent = '.timestamp { cursor: help; }';
        document.head.appendChild(style);
        
        console.log('Time tooltip event delegation setup complete');
    }
    
    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function() {
            initializeTooltip();
            setupEventDelegation();
        });
    } else {
        initializeTooltip();
        setupEventDelegation();
    }
    
    // Export for debugging
    window.timeTooltipDebug = {
        tooltip: () => tooltip,
        currentTarget: () => currentTarget,
        reinit: () => {
            initializeTooltip();
            setupEventDelegation();
        }
    };
})();