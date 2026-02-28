<script lang="ts">
  import PortfolioForm from '$gen/components/entities/portfolio/PortfolioForm.svelte';
  import { portfolioApi } from '$gen/api/portfolio.api';

  async function handleSubmit(event: CustomEvent<{ values: any; mode: string }>) {
    try {
      const portfolio = await portfolioApi.create(event.detail.values);
      window.location.hash = `/portfolios/${portfolio.id}`;
    } catch (e: any) {
      console.error('Create failed:', e);
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/portfolios" class="btn btn-sm variant-soft">&larr; Back to Portfolios</a>
    <h1 class="h2">New Portfolio</h1>
  </div>
  <div class="card p-6">
    <PortfolioForm on:submit={handleSubmit} />
  </div>
</div>
