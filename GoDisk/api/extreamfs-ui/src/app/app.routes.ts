import { Routes } from '@angular/router';
import { Consola } from './features/consola/consola';
import { LoginComponent } from './features/login/login';
import { AuthGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: '', component: Consola, canActivate: [AuthGuard] },
  { path: '**', redirectTo: '' }
];
