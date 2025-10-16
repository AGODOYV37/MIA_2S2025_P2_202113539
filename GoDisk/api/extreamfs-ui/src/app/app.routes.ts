import { Routes } from '@angular/router';
import { Consola } from './features/consola/consola';

export const routes: Routes = [
  { path: '', component: Consola},
  { path: '**', redirectTo: '' }
];
