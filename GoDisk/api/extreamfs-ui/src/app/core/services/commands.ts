import { Injectable , inject} from '@angular/core';
import { HttpClient } from '@angular/common/http';

export interface ExecResponse { output: string; }

@Injectable({
  providedIn: 'root'
})
export class Commands {

  private http = inject(HttpClient);
  private base = '/api'; // usa tu proxy.conf.json

  execute(script: string) {
  return this.http.post<ExecResponse>(`${this.base}/exec`, { script });
  
  }
}
