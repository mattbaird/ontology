<script lang="ts">
  import BuildingForm from '$gen/components/entities/building/BuildingForm.svelte';
  import { buildingApi } from '$gen/api/building.api';

  async function handleSubmit(event: CustomEvent<{ values: any; mode: string }>) {
    try {
      const building = await buildingApi.create(event.detail.values);
      window.location.hash = `/buildings/${building.id}`;
    } catch (e: any) {
      console.error('Create failed:', e);
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/buildings" class="btn btn-sm variant-soft">&larr; Back to Buildings</a>
    <h1 class="h2">New Building</h1>
  </div>
  <div class="card p-6">
    <BuildingForm on:submit={handleSubmit} />
  </div>
</div>
