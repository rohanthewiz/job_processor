								(function() {
// This is an enhanced version of periodic_job_row.js that colors the chart based on job status
// Replace your existing periodicJobRow variable content with this

									const chartContainer = document.querySelector('#chart-' + jobID);
									if (!chartContainer) {
										console.error('Chart container not found for job:', jobID);
										return;
									}

// Get success rate container
									const successRateContainer = document.querySelector('#success-rate-' + jobID);

									fetch('/jobs/history/' + jobID)
										.then(response => response.json())
										.then(data => {
											// Check if the response is an error object
											if (data && data.error) {
												console.error('Error from server:', data.error);
												if (successRateContainer) {
													successRateContainer.innerHTML = '<span style="color: #ef4444;">Error</span>';
												}
												return;
											}
											
											// Check if data is not an array
											if (!Array.isArray(data)) {
												console.error('Expected array but got:', typeof data, data);
												if (successRateContainer) {
													successRateContainer.innerHTML = '<span style="color: #ef4444;">Invalid data</span>';
												}
												return;
											}
											
											if (data.length === 0) {
												if (successRateContainer) {
													successRateContainer.innerHTML = '<span style="color: #666;">No data</span>';
												}
												return;
											}

											// Sort data by time (newest first)
											data.sort((a, b) => new Date(b.StartTime) - new Date(a.StartTime));

											// Take only the last 20 runs for the chart
											const chartData = data.slice(0, 20).reverse();

											// Calculate success rate
											const successCount = data.filter(d => d.Status === 'complete').length;
											const successRate = Math.round((successCount / data.length) * 100);

											// Update success rate display
											if (successRateContainer) {
												const rateColor = successRate >= 80 ? '#22c55e' : successRate >= 50 ? '#f59e0b' : '#ef4444';
												successRateContainer.innerHTML =
													'<div style="font-weight: 600; color: ' + rateColor + ';">' + successRate + '%</div>' +
													'<div style="font-size: 0.7rem; color: #999;">success</div>';
											}

											// Prepare data for Chart.js
											const labels = chartData.map((d, i) => {
												const date = new Date(d.StartTime);
												return date.toLocaleTimeString('en-US', {
													hour: '2-digit',
													minute: '2-digit',
													hour12: false
												});
											});

											const durations = chartData.map(d => d.Duration / 1000000); // Convert to milliseconds

											// Create gradient colors based on status
											const colors = chartData.map(d => {
												if (d.Status === 'complete') {
													return 'rgba(34, 197, 94, 0.8)'; // Green for success
												} else {
													return 'rgba(239, 68, 68, 0.8)'; // Red for failure
												}
											});

											// Create point colors (darker version for points)
											const pointColors = chartData.map(d => {
												if (d.Status === 'complete') {
													return 'rgb(34, 197, 94)'; // Green
												} else {
													return 'rgb(239, 68, 68)'; // Red
												}
											});

											// Create the chart
											const ctx = chartContainer.getContext('2d');

											// Create gradient for the area fill
											const gradient = ctx.createLinearGradient(0, 0, 0, 60);

											// Determine overall color based on recent failures
											const recentRuns = chartData.slice(-5); // Last 5 runs
											const recentFailures = recentRuns.filter(d => d.Status !== 'complete').length;

											if (recentFailures > 2) {
												// Mostly failures - red gradient
												gradient.addColorStop(0, 'rgba(239, 68, 68, 0.3)');
												gradient.addColorStop(1, 'rgba(239, 68, 68, 0.05)');
											} else if (recentFailures > 0) {
												// Some failures - orange gradient
												gradient.addColorStop(0, 'rgba(251, 146, 60, 0.3)');
												gradient.addColorStop(1, 'rgba(251, 146, 60, 0.05)');
											} else {
												// All success - green gradient
												gradient.addColorStop(0, 'rgba(34, 197, 94, 0.3)');
												gradient.addColorStop(1, 'rgba(34, 197, 94, 0.05)');
											}

											new Chart(ctx, {
												type: 'line',
												data: {
													labels: labels,
													datasets: [{
														data: durations,
														borderColor: function(context) {
															const index = context.dataIndex;
															if (index === undefined) return 'rgb(34, 197, 94)';
															return pointColors[index];
														},
														backgroundColor: gradient,
														pointBackgroundColor: pointColors,
														pointBorderColor: pointColors,
														pointRadius: 3,
														pointHoverRadius: 5,
														borderWidth: 2,
														tension: 0.3,
														fill: true,
														segment: {
															borderColor: function(context) {
																// Color line segments based on the status of the end point
																const p1Index = context.p1DataIndex;
																if (p1Index === undefined) return 'rgb(34, 197, 94)';

																// Check if either endpoint is a failure
																const p0Status = chartData[context.p0DataIndex]?.Status;
																const p1Status = chartData[p1Index]?.Status;

																if (p0Status !== 'complete' || p1Status !== 'complete') {
																	return 'rgba(239, 68, 68, 0.8)'; // Red for failure segments
																}
																return 'rgba(34, 197, 94, 0.8)'; // Green for success segments
															}
														}
													}]
												},
												options: {
													responsive: true,
													maintainAspectRatio: false,
													interaction: {
														intersect: false,
														mode: 'index'
													},
													plugins: {
														legend: {
															display: false
														},
														tooltip: {
															backgroundColor: 'rgba(0, 0, 0, 0.8)',
															titleColor: 'white',
															bodyColor: 'white',
															borderColor: 'rgba(255, 255, 255, 0.1)',
															borderWidth: 1,
															padding: 8,
															displayColors: false,
															callbacks: {
																title: function(context) {
																	const index = context[0].dataIndex;
																	const run = chartData[index];
																	return 'Run #' + run.RunNumber + ' - ' + labels[index];
																},
																label: function(context) {
																	const index = context.dataIndex;
																	const run = chartData[index];
																	return [
																		'Duration: ' + context.parsed.y.toFixed(1) + ' ms',
																		'Status: ' + run.Status + (run.Status !== 'complete' ? ' ❌' : ' ✅')
																	];
																},
																labelTextColor: function(context) {
																	const index = context.dataIndex;
																	const run = chartData[index];
																	return run.Status === 'complete' ? '#22c55e' : '#ef4444';
																}
															}
														}
													},
													scales: {
														x: {
															display: false
														},
														y: {
															display: false,
															beginAtZero: true
														}
													}
												}
											});
										})
										.catch(error => {
											console.error('Error fetching job history:', error);
											if (successRateContainer) {
												successRateContainer.innerHTML = '<span style="color: #ef4444;">Error</span>';
											}
										});								})();
