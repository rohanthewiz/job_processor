								(function() {
									const chartId = 'chart-'+jobID;
									const canvas = document.getElementById(chartId);
									if (!canvas) return;

									// Fetch job history
									fetch('/jobs/history/'+jobID)
										.then(response => response.json())
										.then(data => {
											if (!data || data.length === 0) {
												canvas.style.display = 'none';
												const successRateContainer = document.getElementById('success-rate-'+jobID);
												if (successRateContainer) {
													successRateContainer.innerHTML = '<span style="color: #999;">No runs</span>';
												}
												return;
											}

											// Calculate success rate
											const successfulRuns = data.filter(d => d.Status === 'complete').length;
											const successRate = Math.round((successfulRuns / data.length) * 100);

											// Display success rate
											const successRateContainer = document.getElementById('success-rate-'+jobID);
											if (successRateContainer) {
												const rateColor = successRate >= 80 ? '#4ade80' : successRate >= 50 ? '#fbbf24' : '#ef4444';
												successRateContainer.innerHTML =
													'<div style="text-align: center;">' +
													'<div style="font-size: 1.1em; font-weight: bold; color: ' + rateColor + ';">' + successRate + '%</div>' +
													'<div style="font-size: 0.8em; color: #666;">success</div>' +
													'</div>';
											}

											// Prepare chart data
											const labels = data.slice().reverse().map((_, idx) => idx + 1);
											const durations = data.slice().reverse().map(d => d.Duration / 1000000); // Convert to ms
											const colors = data.slice().reverse().map(d =>
												d.Status === 'complete' ? '#4ade80' : '#ef4444'
											);

											// Create chart
											new Chart(canvas, {
												type: 'line',
												data: {
													labels: labels,
													datasets: [{
														data: durations,
														fill: true,
														backgroundColor: 'rgba(74, 222, 128, 0.2)',
														borderColor: '#4ade80',
														borderWidth: 2,
														pointBackgroundColor: colors,
														pointBorderColor: colors,
														pointRadius: 4,
														pointHoverRadius: 6,
														tension: 0.3
													}]
												},
												options: {
													responsive: true,
													maintainAspectRatio: false,
													plugins: {
														legend: { display: false },
														tooltip: {
															callbacks: {
																label: function(context) {
																	return context.parsed.y.toFixed(1) + ' ms';
																}
															}
														}
													},
													scales: {
														x: {
															display: false,
															grid: { display: false }
														},
														y: {
															display: false,
															grid: { display: false },
															beginAtZero: true
														}
													}
												}
											});
										})
										.catch(error => {
											console.error('Error fetching job history:', error);
											canvas.style.display = 'none';
										});
								})();
