# TODOs


## Plan: Limiting Periodic Job Results Display

### Context
After implementing collapsible job results, we need to limit the number of results shown for periodic jobs to maintain performance and UI cleanliness.

### 1. Initial Display Limit
- Show only the last 5-10 results by default when expanded
- Add a "Show more..." / "Show all" button at the bottom of the results
- Display a count indicator: "Showing 5 of 127 results"

### 2. Progressive Loading Options

**Option A: Load More Button** (Recommended to start)
- Add a "Load 10 more" button after the initial results
- Keep loading more in batches as user clicks
- Change to "Show all remaining (X)" when close to the end

**Option B: Pagination**
- Add pagination controls: "< Previous | 1 2 3 ... 10 | Next >"
- Allow jumping to specific pages
- Show 10-20 results per page

**Option C: Smart Grouping**
- Show last 5 results individually
- Group older results by time period: "Last hour (12 runs)", "Last day (288 runs)"
- Expand groups on click

### 3. Result Filtering/Search
- Add a small filter dropdown above results: "All | Successful | Failed"
- Add a date range picker: "Last hour | Last 24h | Last week | Custom"
- Optional: Add a quick search box for error messages

### 4. Visual Enhancements
- Add a subtle gradient fade at the bottom of the result list
- Use alternating row colors for better readability
- Add a thin scrollbar for the results section with max-height
- Show a summary bar: "95% success rate | 5 failures in last 100 runs"

### 5. Performance Considerations
- Load initial results with the main job data
- Lazy-load additional results only when requested
- Consider virtual scrolling for very large result sets
- Cache results in memory to avoid repeated fetches

### 6. UI Integration Example
```
[▼] Periodic Job 1         | ... | [Chart] | [Controls]
    ├─ Summary: Showing 5 of 127 results (95% success)
    ├─ Run #127 | 2024-01-07 10:15:00 | 45ms | ✓
    ├─ Run #126 | 2024-01-07 10:00:00 | 43ms | ✓
    ├─ Run #125 | 2024-01-07 09:45:00 | 52ms | ✗ Error: timeout
    ├─ Run #124 | 2024-01-07 09:30:00 | 41ms | ✓
    ├─ Run #123 | 2024-01-07 09:15:00 | 44ms | ✓
    └─ [Show 10 more ▼] [Show all (122) ▽] [Filter ⚙]
```

### 7. Configuration Options
- Make the default limit configurable (via CLAUDE.md or config)
- Allow users to set their preference (store in localStorage)
- Option to remember expanded/collapsed state per job

### 8. Responsive Behavior
- On mobile: Show only 3-5 results initially
- Reduce columns shown (hide less important fields)
- Make the load more button larger and touch-friendly

### Implementation Notes
- Start with Option A (Load More) as it's simpler while providing good UX
- The current implementation already has the toggle functionality
- Need to modify the SQL query in `GetJobRuns` to support pagination/limits
- May need to add a new endpoint for fetching additional results