<script lang="ts">
  import LeaseList from '$gen/components/entities/lease/LeaseList.svelte';
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="h2">Leases</h1>
    <a href="#/leases/new" class="btn variant-filled-primary">New Lease</a>
  </div>
  <LeaseList />
</div>
