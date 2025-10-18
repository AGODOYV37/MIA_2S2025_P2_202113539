import { Routes } from '@angular/router';
import { Consola } from './features/consola/consola';
import { LoginComponent } from './features/login/login';


export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: '', component: Consola},
  { path: '**', redirectTo: '' }
];
