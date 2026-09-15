<script setup>
const categories = [
  {
    name: 'Group Reports',
    items: ['Group Performance', 'Group Activity', 'Group Growth']
  },
  {
    name: 'Financial Reports',
    items: ['Savings', 'Shares', 'Social Fund', 'Loans', 'Fines']
  },
  {
    name: 'Operations',
    items: ['Meetings', 'Attendance', 'Cycles']
  },
  {
    name: 'Communication',
    items: ['SMS Usage', 'SMS Delivery']
  }
]
</script>

<template>
  <div>
    <div class="page-head">
      <div>
        <h1>Reports</h1>
        <p>Generate reports directly from recorded transactions — nothing is re-entered.</p>
      </div>
    </div>

    <div class="grid2">
      <div>
        <div v-for="cat in categories" :key="cat.name" class="card">
          <div class="card-head"><h3>{{ cat.name }}</h3></div>
          <div v-for="item in cat.items" :key="item" class="kv">
            <span class="k">{{ item }}</span>
            <span style="color:var(--green-600);font-weight:700;font-size:12px;cursor:pointer;">Generate &rarr;</span>
          </div>
        </div>
      </div>
      <div>
        <div class="card">
          <div class="card-head"><h3>Report configuration</h3></div>
          <div class="field-view">
            <label>Date range</label>
            <div class="box">01 Sep 2026 &rarr; 12 Sep 2026</div>
          </div>
          <div class="field-view">
            <label>Group</label>
            <div class="box">All Groups</div>
          </div>
          <button class="btn btn-primary btn-block">Generate report</button>
        </div>
        <div class="card">
          <div class="card-head"><h3>Export</h3></div>
          <div style="display:flex;gap:10px;">
            <button class="btn btn-outline">PDF</button>
            <button class="btn btn-outline">Excel</button>
            <button class="btn btn-outline">CSV</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>