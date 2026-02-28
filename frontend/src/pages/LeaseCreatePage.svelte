<script lang="ts">
  import LeaseForm from '$gen/components/entities/lease/LeaseForm.svelte';
  import { leaseApi } from '$gen/api/lease.api';

  async function handleSubmit(event: CustomEvent<{ values: any; mode: string }>) {
    try {
      const lease = await leaseApi.create(event.detail.values);
      window.location.hash = `/leases/${lease.id}`;
    } catch (e: any) {
      console.error('Create failed:', e);
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/leases" class="btn btn-sm variant-soft">&larr; Back to Leases</a>
    <h1 class="h2">New Lease</h1>
  </div>
  <div class="card p-6">
    <LeaseForm on:submit={handleSubmit} />
  </div>
</div>
