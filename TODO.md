# TODOs

## Implementation Complete! ✅

### All Features Successfully Implemented:

1. ✅ **Backend pagination support**
   - Added `GetJobRunsWithPagination()` to fetch limited results per job with proper main row handling
   - Added `GetJobResultsPaginated()` for loading more results with offset support
   - Modified interfaces and store to support the new pagination methods

2. ✅ **Frontend result limiting (10 by default)**
   - Shows only 10 results by default for periodic jobs
   - Added result count indicator "(showing X of Y)" for periodic jobs
   - Result rows numbered with #1, #2, etc. for better readability

3. ✅ **Load More functionality**
   - Added JavaScript `loadMoreResults()` function
   - Load more buttons appear when job has more than 10 results
   - Button shows "Load X more (showing Y of Z)" with accurate counts
   - New results are inserted seamlessly when load more is clicked

4. ✅ **Success rate in chart area**
   - Added success rate percentage display next to charts for periodic jobs
   - Color coded: green (≥80%), yellow (≥50%), red (<50%)
   - Shows "No runs" for jobs without history
   - Positioned nicely to the left of the mini chart

### Technical Details:
- Fixed rweb query parameter handling using `ctx.Request().QueryParam("offset")`
- Restructured SQL query to ensure all jobs get main rows using UNION approach
- Proper handling of NULL values for main job rows vs result rows
- Clean separation of concerns between job metadata and execution results

### Testing Confirmed:
- Periodic jobs show exactly 10 results initially with accurate count
- Load more button works correctly and loads next 10 results
- Success rate displays correctly with proper color coding
- All jobs (periodic and one-time) render properly with their main rows

### Bug Fix (Chart Overflow):
- Fixed issue where periodic job chart was stretching into the controls column
- Added max-width constraints to chart container (500px inline, 600px CSS)
- Added overflow: hidden to prevent content spillover
- Set controls column to fixed width (150px) to maintain layout
- Added table-layout: auto for better column sizing

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