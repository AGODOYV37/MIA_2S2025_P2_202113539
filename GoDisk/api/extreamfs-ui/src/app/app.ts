import { Component, signal, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { NgIf, AsyncPipe } from '@angular/common';
import { RouterLink, Router } from '@angular/router';
import { AuthService } from './core/services/auth';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, NgIf, AsyncPipe, RouterLink],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App {
  protected readonly title = signal('extreamfs-ui');
  readonly auth = inject(AuthService);
  private router = inject(Router);

  logout() {
    this.auth.logout().subscribe({
      complete: () => this.router.navigateByUrl('/login')
    });
  }
}
