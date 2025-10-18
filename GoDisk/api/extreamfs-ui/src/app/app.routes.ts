import { Routes } from '@angular/router';
import { Consola } from './features/consola/consola';
import { LoginComponent } from './features/login/login';
import { VisualizadorComponent } from './features/visualizador/discos';


export const routes: Routes = [
  { path: '', component: Consola},
  { path: 'login', component: LoginComponent },
  { path: 'visualizador', component: VisualizadorComponent },          
  { path: 'visualizador/:disk', component: VisualizadorComponent }, 
  { path: '**', redirectTo: '' }
];
