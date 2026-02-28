<script lang="ts">
  import OrganizationForm from '$gen/components/entities/organization/OrganizationForm.svelte';
  import { organizationApi } from '$gen/api/organization.api';

  async function handleSubmit(event: CustomEvent<{ values: any; mode: string }>) {
    try {
      const org = await organizationApi.create(event.detail.values);
      window.location.hash = `/organizations/${org.id}`;
    } catch (e: any) {
      console.error('Create failed:', e);
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/organizations" class="btn btn-sm variant-soft">&larr; Back to Organizations</a>
    <h1 class="h2">New Organization</h1>
  </div>
  <div class="card p-6">
    <OrganizationForm on:submit={handleSubmit} />
  </div>
</div>
