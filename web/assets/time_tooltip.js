// Time tooltip functionality
(function() {
    'use strict';
    
    let tooltip;
    
    // Initialize tooltip when DOM is ready
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
    
    // Show tooltip
    function showTooltip(e) {
        if (!tooltip) return; // Tooltip not initialized yet
        
        const target = e.currentTarget;
        const utcTime = target.textContent.trim();
        const localTime = convertToLocalTime(utcTime);
        
        if (!localTime) return;
        
        tooltip.innerHTML = `
            <div style="margin-bottom: 4px;"><strong>UTC:</strong> ${localTime.utc}</div>
            <div><strong>Local:</strong> ${localTime.local}</div>
            <div style="font-size: 12px; opacity: 0.8; margin-top: 4px;">Timezone: ${localTime.timezone}</div>
        `;
        
        const rect = target.getBoundingClientRect();
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
        }
    }
    
    // Initialize tooltips for existing elements
    function initTooltips() {
        initializeTooltip(); // Ensure tooltip is created
        
        const timestamps = document.querySelectorAll('.timestamp');
        console.log('Initializing tooltips for', timestamps.length, 'timestamp elements');
        
        timestamps.forEach(elem => {
            elem.removeEventListener('mouseenter', showTooltip);
            elem.removeEventListener('mouseleave', hideTooltip);
            elem.addEventListener('mouseenter', showTooltip);
            elem.addEventListener('mouseleave', hideTooltip);
            elem.style.cursor = 'help';
        });
    }
    
    // Initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function() {
            initTooltips();
            setupEventListeners();
        });
    } else {
        initTooltips();
        setupEventListeners();
    }
    
    function setupEventListeners() {
        // Re-initialize when HTMX updates content
        document.body.addEventListener('htmx:afterSwap', function(event) {
            console.log('HTMX afterSwap event fired', event.detail);
            setTimeout(initTooltips, 100);
        });
        
        // Also listen for the afterSettle event which fires after htmx has settled
        document.body.addEventListener('htmx:afterSettle', function(event) {
            console.log('HTMX afterSettle event fired', event.detail);
            setTimeout(initTooltips, 100);
        });
        
        // Also listen for the SSE events that might update the table
        document.body.addEventListener('htmx:sseMessage', function(event) {
            console.log('HTMX SSE message received', event.detail);
            setTimeout(initTooltips, 100);
        });
        
        // Direct observation of the jobs table body
        setTimeout(function() {
            const jobsTableBody = document.getElementById('jobs-table-body');
            if (jobsTableBody) {
                console.log('Found jobs-table-body, observing for changes');
                const tableObserver = new MutationObserver(function(mutations) {
                    console.log('Jobs table body mutated');
                    setTimeout(initTooltips, 100);
                });
                
                tableObserver.observe(jobsTableBody, {
                    childList: true,
                    subtree: true
                });
            }
        }, 500);
    }
})();