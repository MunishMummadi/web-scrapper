// Main JavaScript file for Web Scraper UI

document.addEventListener('DOMContentLoaded', function() {
  // Initialize charts if they exist on the page
  initializeCharts();
  
  // Initialize form submission handlers
  initializeFormHandlers();
  
  // Initialize table sorting and filtering
  initializeDataTables();
  
  // Initialize data refresh for dashboard stats
  initializeDataRefresh();
});

// Chart initialization
function initializeCharts() {
  // Only initialize charts if the necessary elements exist
  if (!document.getElementById('scraping-activity-chart')) return;
  
  // Activity Chart (using Chart.js)
  const activityCanvas = document.getElementById('scraping-activity-chart').getContext('2d');
  const activityChart = new Chart(activityCanvas, {
    type: 'line',
    data: {
      labels: getLast7Days(),
      datasets: [{
        label: 'Pages Scraped',
        data: [65, 78, 52, 91, 43, 87, 99],
        borderColor: '#2563eb',
        backgroundColor: 'rgba(37, 99, 235, 0.1)',
        borderWidth: 2,
        tension: 0.3,
        fill: true
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: {
          beginAtZero: true,
          grid: {
            display: true,
            color: 'rgba(0, 0, 0, 0.05)'
          }
        },
        x: {
          grid: {
            display: false
          }
        }
      },
      plugins: {
        legend: {
          display: false
        }
      }
    }
  });
  
  // Domain Distribution Chart
  const domainsCanvas = document.getElementById('domain-distribution-chart').getContext('2d');
  const domainsChart = new Chart(domainsCanvas, {
    type: 'doughnut',
    data: {
      labels: ['example.com', 'test.org', 'sample.net', 'demo.io', 'Other'],
      datasets: [{
        data: [35, 25, 15, 10, 15],
        backgroundColor: [
          '#2563eb',
          '#4f46e5',
          '#06b6d4',
          '#10b981',
          '#64748b'
        ],
        borderWidth: 0
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: 'right'
        }
      }
    }
  });
  
  // Performance Chart
  const performanceCanvas = document.getElementById('performance-chart').getContext('2d');
  const performanceChart = new Chart(performanceCanvas, {
    type: 'bar',
    data: {
      labels: getLast7Days(),
      datasets: [{
        label: 'Avg. Scrape Time (ms)',
        data: [120, 132, 101, 134, 90, 110, 125],
        backgroundColor: '#4f46e5'
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: {
          beginAtZero: true,
          grid: {
            display: true,
            color: 'rgba(0, 0, 0, 0.05)'
          }
        },
        x: {
          grid: {
            display: false
          }
        }
      }
    }
  });
}

// Form handlers initialization
function initializeFormHandlers() {
  const urlForm = document.getElementById('url-form');
  if (!urlForm) return;
  
  urlForm.addEventListener('submit', function(e) {
    e.preventDefault();
    const urlInput = document.getElementById('url-input');
    const submitButton = document.getElementById('url-submit');
    const statusMessage = document.getElementById('status-message');
    
    if (!urlInput.value) {
      showFormError(statusMessage, 'Please enter a URL');
      return;
    }
    
    // Show loading state
    submitButton.disabled = true;
    submitButton.innerHTML = '<span class="loader"></span> Processing';
    statusMessage.textContent = '';
    
    // Make API request to enqueue URL
    fetch('/api/enqueue', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: `url=${encodeURIComponent(urlInput.value)}`
    })
    .then(response => {
      if (!response.ok) {
        throw new Error(`HTTP error ${response.status}`);
      }
      return response.text();
    })
    .then(data => {
      showFormSuccess(statusMessage, 'URL successfully queued for scraping');
      urlInput.value = '';
    })
    .catch(error => {
      showFormError(statusMessage, `Error: ${error.message}`);
    })
    .finally(() => {
      // Reset button state
      submitButton.disabled = false;
      submitButton.innerHTML = 'Scrape URL';
    });
  });
}

// Initialize DataTables for better table interactions
function initializeDataTables() {
  const scrapedDataTable = document.getElementById('scraped-data-table');
  if (!scrapedDataTable) return;
  
  // Simple client-side filtering
  const searchInput = document.getElementById('table-search');
  if (searchInput) {
    searchInput.addEventListener('input', function() {
      const searchTerm = this.value.toLowerCase();
      const tableRows = scrapedDataTable.querySelectorAll('tbody tr');
      
      tableRows.forEach(row => {
        const text = row.textContent.toLowerCase();
        row.style.display = text.includes(searchTerm) ? '' : 'none';
      });
    });
  }
  
  // Simple client-side sorting
  const sortableHeaders = scrapedDataTable.querySelectorAll('th[data-sort]');
  sortableHeaders.forEach(header => {
    header.addEventListener('click', function() {
      const sortKey = this.dataset.sort;
      const isAsc = this.classList.contains('sort-asc');
      
      // Update header state
      sortableHeaders.forEach(h => h.classList.remove('sort-asc', 'sort-desc'));
      this.classList.add(isAsc ? 'sort-desc' : 'sort-asc');
      
      // Sort table rows
      const tableBody = scrapedDataTable.querySelector('tbody');
      const rows = Array.from(tableBody.querySelectorAll('tr'));
      
      rows.sort((a, b) => {
        const aValue = a.querySelector(`td[data-${sortKey}]`).dataset[sortKey];
        const bValue = b.querySelector(`td[data-${sortKey}]`).dataset[sortKey];
        
        if (sortKey === 'date') {
          return isAsc ? 
            new Date(bValue) - new Date(aValue) : 
            new Date(aValue) - new Date(bValue);
        }
        
        return isAsc ? 
          bValue.localeCompare(aValue) : 
          aValue.localeCompare(bValue);
      });
      
      // Reappend sorted rows
      rows.forEach(row => tableBody.appendChild(row));
    });
  });
}

// Initialize periodic data refresh for dashboard
function initializeDataRefresh() {
  const dashboardStats = document.getElementById('dashboard-stats');
  if (!dashboardStats) return;
  
  // Refresh dashboard stats every 30 seconds
  setInterval(() => {
    fetch('/api/stats')
      .then(response => response.json())
      .then(data => {
        updateDashboardStats(data);
      })
      .catch(error => console.error('Error fetching stats:', error));
  }, 30000);
}

// Helper to update dashboard stats
function updateDashboardStats(data) {
  const elements = {
    totalUrls: document.getElementById('total-urls-stat'),
    queuedUrls: document.getElementById('queued-urls-stat'),
    crawlRate: document.getElementById('crawl-rate-stat'),
    errorRate: document.getElementById('error-rate-stat')
  };
  
  if (elements.totalUrls) elements.totalUrls.textContent = data.totalUrls;
  if (elements.queuedUrls) elements.queuedUrls.textContent = data.queuedUrls;
  if (elements.crawlRate) elements.crawlRate.textContent = data.crawlRate;
  if (elements.errorRate) elements.errorRate.textContent = data.errorRate + '%';
}

// Helper to show form success message
function showFormSuccess(element, message) {
  element.textContent = message;
  element.className = 'form-message success';
  setTimeout(() => {
    element.textContent = '';
    element.className = 'form-message';
  }, 5000);
}

// Helper to show form error message
function showFormError(element, message) {
  element.textContent = message;
  element.className = 'form-message error';
}

// Helper to get the last 7 days for charts
function getLast7Days() {
  const days = [];
  for (let i = 6; i >= 0; i--) {
    const date = new Date();
    date.setDate(date.getDate() - i);
    days.push(date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));
  }
  return days;
}

// Toggle mobile menu
function toggleMobileMenu() {
  const mobileMenu = document.getElementById('mobile-menu');
  mobileMenu.classList.toggle('open');
}

// Show confirmation dialog
function confirmAction(message, callback) {
  if (confirm(message)) {
    callback();
  }
}

// Handle URL submission through the API
function submitUrl(url) {
  return fetch('/api/enqueue', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: `url=${encodeURIComponent(url)}`
  })
  .then(response => {
    if (!response.ok) {
      throw new Error(`HTTP error ${response.status}`);
    }
    return response.text();
  });
}

// Fetch scraped data from API
function fetchScrapedData(limit = 100, page = 1) {
  return fetch(`/api/data?limit=${limit}&page=${page}`)
    .then(response => {
      if (!response.ok) {
        throw new Error(`HTTP error ${response.status}`);
      }
      return response.json();
    });
}
