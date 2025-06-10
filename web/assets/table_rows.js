	function toggleJobResults(jobId) {
		const resultRows = document.querySelectorAll('.job-result-row[data-job-id="' + jobId + '"]');
		const loadMoreRows = document.querySelectorAll('.load-more-row.job-' + jobId);
		const toggleBtn = document.querySelector('.toggle-btn[data-job-id="' + jobId + '"]');
		const isExpanded = toggleBtn.classList.contains('expanded');

		// Toggle result rows
		resultRows.forEach(row => {
			row.style.display = isExpanded ? 'none' : '';
		});

		// Toggle load more rows
		loadMoreRows.forEach(row => {
			row.style.display = isExpanded ? 'none' : '';
		});

		toggleBtn.classList.toggle('expanded');
		toggleBtn.textContent = isExpanded ? '▶' : '▼';

		// Store state in localStorage
		const expandedJobs = JSON.parse(localStorage.getItem('expandedJobs') || '{}');
		expandedJobs[jobId] = !isExpanded;
		localStorage.setItem('expandedJobs', JSON.stringify(expandedJobs));
	}

	// Restore expanded state after HTMX update
	document.addEventListener('htmx:afterSwap', function() {
		const expandedJobs = JSON.parse(localStorage.getItem('expandedJobs') || '{}');
		Object.entries(expandedJobs).forEach(([jobId, isExpanded]) => {
			if (isExpanded) {
				const toggleBtn = document.querySelector('.toggle-btn[data-job-id="' + jobId + '"]');
				if (toggleBtn && !toggleBtn.classList.contains('expanded')) {
					toggleJobResults(jobId);
				}
			}
		});

		// Ensure all flex containers are properly styled
		const allFlexContainers = document.querySelectorAll('td > div[style*="flex"]');
		allFlexContainers.forEach(container => {
			// Force re-apply flex styles in case they got lost
			container.style.display = 'flex';
			container.style.alignItems = 'center';
			container.style.gap = '0.5rem';
		});
	});

	function loadMoreResults(button) {
		const jobId = button.getAttribute('data-job-id');
		const offset = button.getAttribute('data-offset');
		const loadMoreRow = button.closest('tr');

		fetch('/jobs/results/' + jobId + '?offset=' + offset)
			.then(response => response.text())
			.then(html => {
				// Insert the new rows before the load more row
				loadMoreRow.insertAdjacentHTML('beforebegin', html);
				// Remove the old load more row
				loadMoreRow.remove();
				// Make sure new rows are visible if job is expanded
				const toggleBtn = document.querySelector('.toggle-btn[data-job-id="' + jobId + '"]');
				if (toggleBtn && toggleBtn.classList.contains('expanded')) {
					const newRows = document.querySelectorAll('.job-result-row[data-job-id="' + jobId + '"], .load-more-row.job-' + jobId);
					newRows.forEach(row => {
						row.style.display = '';
					});
				}
			})
			.catch(error => {
				console.error('Error loading more results:', error);
				button.textContent = 'Error loading results';
				button.disabled = true;
			});
	}
