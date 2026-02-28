<script lang="ts">
  import PropertyForm from '$gen/components/entities/property/PropertyForm.svelte';
  import { propertyApi } from '$gen/api/property.api';

  async function handleSubmit(event: CustomEvent<{ values: any; mode: string }>) {
    try {
      const property = await propertyApi.create(event.detail.values);
      window.location.hash = `/properties/${property.id}`;
    } catch (e: any) {
      console.error('Create failed:', e);
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/properties" class="btn btn-sm variant-soft">&larr; Back to Properties</a>
    <h1 class="h2">New Property</h1>
  </div>
  <div class="card p-6">
    <PropertyForm on:submit={handleSubmit} />
  </div>
</div>
