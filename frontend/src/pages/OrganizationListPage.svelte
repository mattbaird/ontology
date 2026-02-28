<script lang="ts">
  import OrganizationList from '$gen/components/entities/organization/OrganizationList.svelte';
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="h2">Organizations</h1>
    <a href="#/organizations/new" class="btn variant-filled-primary">New Organization</a>
  </div>
  <OrganizationList />
</div>
