import { RenderMode, ServerRoute } from '@angular/ssr';

export const serverRoutes: ServerRoute[] = [
  // ✅ SOLO estas rutas estáticas se prerenderán:
  { path: '',              renderMode: RenderMode.Prerender },
  { path: 'login',         renderMode: RenderMode.Prerender },

  // 🚫 NO prerender para páginas que hacen llamadas a /api:
  { path: 'visualizador',        renderMode: RenderMode.Server },
  { path: 'visualizador/:id',    renderMode: RenderMode.Server },
  { path: 'visualizador/:id/fs', renderMode: RenderMode.Server },

  // Wildcard sin prerender
  { path: '**',            renderMode: RenderMode.Server },
];
    