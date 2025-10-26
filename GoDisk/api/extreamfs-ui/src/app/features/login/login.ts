import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router} from '@angular/router';
import { AuthService } from '../../core/services/auth';
import { Commands } from '../../core/services/commands';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './login.html',
  styleUrls: ['./login.scss']
})
export class LoginComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private cmd = inject(Commands);

  usr = '';
  pwd = '';
  mountId = '';          
  cargando = false;
  error = '';
  mountedRaw = '';       

  submit() {
    if (!this.usr.trim() || !this.pwd.trim() || !this.mountId.trim()) {
      this.error = 'Completa usuario, contraseña e ID de partición.';
      return;
    }
    this.cargando = true;
    this.error = '';

    this.auth.login(this.usr.trim(), this.pwd, this.mountId.trim())
      .subscribe({
        next: () => {
          this.router.navigateByUrl('/');
        },
        error: (e: any) => {
          this.error = (e?.message || 'Fallo de autenticación').trim();
        },
        complete: () => { this.cargando = false; }
      });
  }


  verMontajes() {
    this.cargando = true;
    this.error = '';

    this.cmd.execute('mounted').subscribe({
      next: res => {
        this.mountedRaw = (res?.output || '').trim() || '(sin particiones montadas)';
      },
      error: err => {
        this.mountedRaw = '';
        this.error = err?.message || 'No se pudo obtener montajes';
      },
      complete: () => { this.cargando = false; }
    });
  }
}
