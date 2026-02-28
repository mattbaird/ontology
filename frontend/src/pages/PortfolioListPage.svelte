<script lang="ts">
  import PortfolioList from '$gen/components/entities/portfolio/PortfolioList.svelte';
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="h2">Portfolios</h1>
    <a href="#/portfolios/new" class="btn variant-filled-primary">New Portfolio</a>
  </div>
  <PortfolioList />
</div>
