import { Routes } from '@angular/router';
import { Consola } from './features/consola/consola';
import { LoginComponent } from './features/login/login';

export const routes: Routes = [
  { path: '', component: Consola },
  { path: 'login', component: LoginComponent },

  {
    path: 'visualizador',
    loadComponent: () =>
      import('./features/visualizador/discos').then(m => m.VisualizadorComponent)
  },

  {
    path: 'visualizador/:id',
    loadComponent: () =>
      import('./features/visualizador/part').then(m => m.VisualizadorPartComponent)
  },

  {
    path: 'visualizador/:id/fs',
    loadComponent: () =>
      import('./features/visualizador/fs').then(m => m.FsExplorerComponent)
  },

  { path: '**', redirectTo: '' }
];
